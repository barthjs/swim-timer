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
}

// Option configures a Checker.
type Option func(*Checker)

// WithOptional registers a dependency the application can run without. Its failure marks the application
// as degraded instead of unavailable. A name that repeats a required or another optional dependency
// would hide one of them in the checks, so NewChecker panics on it.
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

// NewChecker creates a Checker. The pingers are required dependencies, see WithOptional for the others.
// It panics if an optional dependency repeats the name of another dependency.
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

// Ready reports whether the application can accept traffic. A failing required dependency answers 503
// with the status unavailable, a failing optional one answers 200 with the status degraded.
// The checks name the state of every dependency.
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
