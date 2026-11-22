// Package health provides readiness and liveness probes behind a standard HTTP handler.
//
// Readiness distinguishes required dependencies, without which the application cannot serve traffic,
// from optional ones it can run without. A failing required dependency reports the application as unavailable (503).
// A failing optional one reports it as degraded, which still answers 200,
// so a load balancer keeps routing traffic while the response names what is missing.
package health

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"maps"
	"net/http"
	"sync"
	"time"
)

// readyTimeout bounds the readiness check. Every dependency gets the whole budget, since they are checked concurrently.
const readyTimeout = 3 * time.Second

// Pinger reports whether a backing dependency is reachable.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Readiness states reported by Ready.
const (
	StatusOK          = "ok"
	StatusDegraded    = "degraded"
	StatusUnavailable = "unavailable"
)

// Check states reported per dependency.
const (
	CheckOK          = "ok"
	CheckUnreachable = "unreachable"
)

// Checker provides the health and readiness endpoints.
type Checker struct {
	version  string
	started  time.Time
	pingers  map[string]Pinger
	optional map[string]Pinger
	onChange func(checks map[string]string)

	// refreshing runs the refreshes one after the other, so they report their states in order.
	refreshing sync.Mutex
	mu         sync.Mutex
	reported   map[string]string
}

// Option configures a Checker.
type Option func(*Checker)

// WithOptional registers a dependency the application can run without, whose failure marks it as degraded.
func WithOptional(name string, pinger Pinger) Option {
	return func(c *Checker) {
		_, required := c.pingers[name]
		_, optional := c.optional[name]
		if required || optional {
			panic(fmt.Sprintf("health: the dependency %q is registered twice", name))
		}
		c.optional[name] = pinger
	}
}

// WithOnChecksChange calls onChange with the state of every dependency once Refresh finds that one of them changed.
func WithOnChecksChange(onChange func(checks map[string]string)) Option {
	return func(c *Checker) {
		c.onChange = onChange
	}
}

// NewChecker creates a Checker for the required pingers and panics on an optional dependency that repeats a name,
// since the checks would hide one of them.
func NewChecker(version string, pingers map[string]Pinger, opts ...Option) *Checker {
	c := &Checker{
		version:  version,
		started:  time.Now(),
		pingers:  pingers,
		optional: make(map[string]Pinger),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Version returns the running build's version string.
func (c *Checker) Version() string {
	return c.version
}

// Uptime returns how long the process has been running.
func (c *Checker) Uptime() time.Duration {
	return time.Since(c.started)
}

// Checks returns the state of every dependency.
func (c *Checker) Checks(ctx context.Context) map[string]string {
	ctx, cancel := context.WithTimeout(ctx, readyTimeout)
	defer cancel()

	checks, _, _ := c.pingAll(ctx)
	return checks
}

// Refresh checks every dependency and reports their states to the callback of WithOnChecksChange
// when they differ from the last reported ones. A dependency calls it once it notices a change of its own,
// so the states are reported without polling.
func (c *Checker) Refresh() {
	c.refreshing.Lock()
	defer c.refreshing.Unlock()

	checks := c.Checks(context.Background())
	c.mu.Lock()
	changed := c.reported == nil || !maps.Equal(checks, c.reported)
	c.reported = checks
	c.mu.Unlock()

	if changed && c.onChange != nil {
		c.onChange(checks)
	}
}

// Reported returns the states the last Refresh found without checking the dependencies again,
// so a caller on a hot path never waits for a ping. It is empty before the first Refresh.
func (c *Checker) Reported() map[string]string {
	c.mu.Lock()
	defer c.mu.Unlock()

	return maps.Clone(c.reported)
}

// status is the JSON body of the health endpoints.
type status struct {
	Status  string            `json:"status"`
	Version string            `json:"version"`
	Uptime  string            `json:"uptime"`
	Checks  map[string]string `json:"checks,omitempty"`
}

// Live indicates whether the application is currently running.
func (c *Checker) Live(w http.ResponseWriter, _ *http.Request) {
	c.write(w, http.StatusOK, status{
		Status:  StatusOK,
		Version: c.version,
		Uptime:  c.Uptime().Round(time.Second).String(),
	})
}

// Ready answers whether the application can accept traffic, with the state of every dependency.
func (c *Checker) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), readyTimeout)
	defer cancel()

	checks, requiredOK, optionalOK := c.pingAll(ctx)

	code := http.StatusOK
	state := StatusOK
	switch {
	case !requiredOK:
		code = http.StatusServiceUnavailable
		state = StatusUnavailable
	case !optionalOK:
		state = StatusDegraded
	}

	c.write(w, code, status{
		Status:  state,
		Version: c.version,
		Checks:  checks,
		Uptime:  c.Uptime().Round(time.Second).String(),
	})
}

// pingAll checks every dependency concurrently, so a slow one cannot use up the budget of the others.
// It returns the state of every dependency and whether all required and all optional ones are reachable.
func (c *Checker) pingAll(ctx context.Context) (map[string]string, bool, bool) {
	var (
		mu         sync.Mutex
		wg         sync.WaitGroup
		requiredOK = true
		optionalOK = true
	)
	checks := make(map[string]string, len(c.pingers)+len(c.optional))

	start := func(name string, pinger Pinger, required bool) {
		if pinger == nil {
			return
		}
		wg.Go(func() {
			err := pinger.Ping(ctx)

			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				checks[name] = CheckOK
				return
			}
			checks[name] = CheckUnreachable
			if required {
				requiredOK = false
			} else {
				optionalOK = false
			}
		})
	}

	for name, pinger := range c.pingers {
		start(name, pinger, true)
	}
	for name, pinger := range c.optional {
		start(name, pinger, false)
	}
	wg.Wait()

	return checks, requiredOK, optionalOK
}

// Handler mounts the probes at /healthz (liveness) and /readyz (readiness) on a fresh http.ServeMux.
func (c *Checker) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", c.Live)
	mux.HandleFunc("/readyz", c.Ready)
	return mux
}

// write encodes the status as JSON with the given HTTP status code.
func (c *Checker) write(w http.ResponseWriter, code int, body status) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.MarshalWrite(w, body)
}
