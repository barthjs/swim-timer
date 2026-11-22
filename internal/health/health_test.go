package health_test

import (
	"context"
	"encoding/json/v2"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/health"
)

// pingerFunc adapts a function to the Pinger interface.
type pingerFunc func(ctx context.Context) error

// Ping calls the function.
func (f pingerFunc) Ping(ctx context.Context) error {
	return f(ctx)
}

// response is the decoded body of a health check.
type response struct {
	Status  string            `json:"status"`
	Version string            `json:"version"`
	Uptime  string            `json:"uptime"`
	Checks  map[string]string `json:"checks"`
}

func TestWithOptional(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		required   map[string]health.Pinger
		optional   map[string]health.Pinger
		wantCode   int
		wantStatus string
		wantChecks map[string]string
	}{
		{
			name:       "reports reachable dependencies as ok",
			required:   map[string]health.Pinger{"db": healthy()},
			optional:   map[string]health.Pinger{"device": healthy()},
			wantCode:   http.StatusOK,
			wantStatus: health.StatusOK,
			wantChecks: map[string]string{"db": health.CheckOK, "device": health.CheckOK},
		},
		{
			name:       "degrades on an unreachable optional dependency",
			required:   map[string]health.Pinger{"db": healthy()},
			optional:   map[string]health.Pinger{"device": failing()},
			wantCode:   http.StatusOK,
			wantStatus: health.StatusDegraded,
			wantChecks: map[string]string{"db": health.CheckOK, "device": health.CheckUnreachable},
		},
		{
			name:       "reports unavailable over degraded",
			required:   map[string]health.Pinger{"db": failing()},
			optional:   map[string]health.Pinger{"device": failing()},
			wantCode:   http.StatusServiceUnavailable,
			wantStatus: health.StatusUnavailable,
			wantChecks: map[string]string{"db": health.CheckUnreachable, "device": health.CheckUnreachable},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var opts []health.Option
			for name, pinger := range tt.optional {
				opts = append(opts, health.WithOptional(name, pinger))
			}
			checker := health.NewChecker("1.2.3", tt.required, opts...)

			code, body := get(t, checker.Handler(), "/readyz")

			assert.Equal(t, tt.wantCode, code)
			assert.Equal(t, tt.wantStatus, body.Status)
			assert.Equal(t, tt.wantChecks, body.Checks)
		})
	}
}

func TestNewChecker(t *testing.T) {
	t.Parallel()

	t.Run("rejects an optional name that repeats a required one", func(t *testing.T) {
		t.Parallel()

		create := func() {
			health.NewChecker("1.2.3", map[string]health.Pinger{"db": healthy()}, health.WithOptional("db", healthy()))
		}

		assert.Panics(t, create)
	})
}

func TestChecker_Version(t *testing.T) {
	t.Parallel()

	checker := health.NewChecker("1.2.3", nil)

	version := checker.Version()

	assert.Equal(t, "1.2.3", version)
}

func TestChecker_Uptime(t *testing.T) {
	t.Parallel()

	checker := health.NewChecker("1.2.3", nil)

	uptime := checker.Uptime()

	assert.GreaterOrEqual(t, uptime, time.Duration(0))
}

func TestChecker_Checks(t *testing.T) {
	t.Parallel()

	t.Run("returns the state of every dependency", func(t *testing.T) {
		t.Parallel()
		checker := health.NewChecker("1.2.3", map[string]health.Pinger{"db": healthy()}, health.WithOptional("cache", failing()))

		checks := checker.Checks(t.Context())

		assert.Equal(t, map[string]string{"db": "ok", "cache": "unreachable"}, checks)
	})
}

func TestChecker_Refresh(t *testing.T) {
	t.Parallel()

	// newWatchedChecker returns a checker of one dependency whose state the test switches, and the reported states
	newWatchedChecker := func() (*health.Checker, *error, *[]map[string]string) {
		var state error
		var reported []map[string]string
		checker := health.NewChecker("1.2.3", nil,
			health.WithOptional("device", pingerFunc(func(context.Context) error { return state })),
			health.WithOnChecksChange(func(checks map[string]string) { reported = append(reported, checks) }),
		)
		return checker, &state, &reported
	}

	t.Run("reports the states of the first refresh", func(t *testing.T) {
		t.Parallel()
		checker, _, reported := newWatchedChecker()

		checker.Refresh()

		assert.Equal(t, []map[string]string{{"device": "ok"}}, *reported)
	})

	t.Run("reports a changed state", func(t *testing.T) {
		t.Parallel()
		checker, state, reported := newWatchedChecker()
		checker.Refresh()
		*state = errors.New("disconnected")

		checker.Refresh()

		assert.Equal(t, []map[string]string{{"device": "ok"}, {"device": "unreachable"}}, *reported)
	})

	t.Run("skips unchanged states", func(t *testing.T) {
		t.Parallel()
		checker, _, reported := newWatchedChecker()
		checker.Refresh()

		checker.Refresh()

		assert.Len(t, *reported, 1)
	})

	t.Run("keeps the states without a callback", func(t *testing.T) {
		t.Parallel()
		checker := health.NewChecker("1.2.3", map[string]health.Pinger{"db": healthy()})

		checker.Refresh()

		assert.Equal(t, map[string]string{"db": "ok"}, checker.Reported())
	})
}

func TestChecker_Reported(t *testing.T) {
	t.Parallel()

	t.Run("returns the states of the last refresh without pinging", func(t *testing.T) {
		t.Parallel()
		pings := 0
		checker := health.NewChecker("1.2.3", map[string]health.Pinger{"db": pingerFunc(func(context.Context) error {
			pings++
			return nil
		})})
		checker.Refresh()

		checks := checker.Reported()

		assert.Equal(t, map[string]string{"db": "ok"}, checks)
		assert.Equal(t, 1, pings)
	})
}

func TestChecker_Live(t *testing.T) {
	t.Parallel()

	checker := health.NewChecker("1.2.3", map[string]health.Pinger{"db": failing()})

	code, body := get(t, checker.Handler(), "/healthz")

	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, "ok", body.Status)
	assert.Equal(t, "1.2.3", body.Version)
	assert.NotEmpty(t, body.Uptime)
	assert.Empty(t, body.Checks, "liveness does not run the dependency checks")
}

func TestChecker_Ready(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		pingers    map[string]health.Pinger
		wantCode   int
		wantStatus string
		wantChecks map[string]string
	}{
		{
			name:       "returns ok when all dependencies are reachable",
			pingers:    map[string]health.Pinger{"db": healthy(), "cache": healthy()},
			wantCode:   http.StatusOK,
			wantStatus: "ok",
			wantChecks: map[string]string{"db": "ok", "cache": "ok"},
		},
		{
			name:       "skips nil pingers",
			pingers:    map[string]health.Pinger{"db": healthy(), "optional": nil},
			wantCode:   http.StatusOK,
			wantStatus: "ok",
			wantChecks: map[string]string{"db": "ok"},
		},
		{
			name:       "returns unavailable when a dependency is unreachable",
			pingers:    map[string]health.Pinger{"db": healthy(), "cache": failing()},
			wantCode:   http.StatusServiceUnavailable,
			wantStatus: "unavailable",
			wantChecks: map[string]string{"db": "ok", "cache": "unreachable"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			checker := health.NewChecker("1.2.3", tt.pingers)

			code, body := get(t, checker.Handler(), "/readyz")

			assert.Equal(t, tt.wantCode, code)
			assert.Equal(t, tt.wantStatus, body.Status)
			assert.Equal(t, len(tt.wantChecks), len(body.Checks))
			for name, want := range tt.wantChecks {
				assert.Equal(t, want, body.Checks[name], name)
			}
		})
	}

	t.Run("bounds the pings with a deadline", func(t *testing.T) {
		t.Parallel()

		slow := pingerFunc(func(ctx context.Context) error {
			_, hasDeadline := ctx.Deadline()
			if !hasDeadline {
				return errors.New("ping without a deadline")
			}
			return nil
		})
		checker := health.NewChecker("1.2.3", map[string]health.Pinger{"slow": slow})

		code, _ := get(t, checker.Handler(), "/readyz")

		assert.Equal(t, http.StatusOK, code)
	})

	t.Run("pings the dependencies concurrently", func(t *testing.T) {
		t.Parallel()

		// Each pinger waits until all of them have started, which only succeeds if they run at the same time
		var started sync.WaitGroup
		started.Add(2)
		allStarted := make(chan struct{})
		go func() {
			started.Wait()
			close(allStarted)
		}()
		waiting := pingerFunc(func(ctx context.Context) error {
			started.Done()
			select {
			case <-allStarted:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
		checker := health.NewChecker("1.2.3", map[string]health.Pinger{"db": waiting}, health.WithOptional("device", waiting))

		code, body := get(t, checker.Handler(), "/readyz")

		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, map[string]string{
			"db":     health.CheckOK,
			"device": health.CheckOK,
		}, body.Checks)
	})
}

func TestChecker_Handler(t *testing.T) {
	t.Parallel()
}

// healthy returns a pinger that always succeeds.
func healthy() health.Pinger {
	return pingerFunc(func(context.Context) error { return nil })
}

// failing returns a pinger that always fails.
func failing() health.Pinger {
	return pingerFunc(func(context.Context) error { return errors.New("connection refused") })
}

// get sends a GET request to the handler and returns the status and the decoded body.
func get(t *testing.T, handler http.Handler, path string) (int, response) {
	t.Helper()

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))

	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

	var body response
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	return recorder.Code, body
}
