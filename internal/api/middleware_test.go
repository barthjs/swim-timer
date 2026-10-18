package api

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
	"github.com/barthjs/swim-timer/internal/ratelimiter"
)

func TestAPI_RecoverPanic(t *testing.T) {
	t.Parallel()

	t.Run("passes the request to the handler", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })

		newBareAPI().recoverPanic(next).ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusNoContent, recorder.Code)
		assert.Empty(t, recorder.Header().Get("Connection"))
	})

	t.Run("answers a panic with a server error and logs it", func(t *testing.T) {
		t.Parallel()

		var logs bytes.Buffer
		a := &API{
			logger: zerolog.New(&logs),
		}
		message := apitest.Faker(t).LetterN(16)
		recorder := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic(message) })

		a.recoverPanic(next).ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
		assert.Equal(t, codeServerError, errorCode(t, recorder))
		assert.Equal(t, "close", recorder.Header().Get("Connection"))
		assert.Contains(t, logs.String(), message)
		assert.Contains(t, logs.String(), "internal server error")
	})
}

// TestAPI_EnforceRateLimit covers the failures of the limiter, which no handler can provoke,
// so the tests replace the limiter instead of driving a handler.
func TestAPI_EnforceRateLimit(t *testing.T) {
	t.Parallel()

	t.Run("allows the request and reports the limit", func(t *testing.T) {
		t.Parallel()

		a := newRateLimitedAPI(t, nil, nil)
		recorder := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)

		allowed := a.enforceRateLimit(recorder, request, apitest.Faker(t).LetterN(16), 2, time.Minute)

		assert.True(t, allowed)
		assert.Equal(t, "2", recorder.Header().Get("RateLimit-Limit"))
		assert.Equal(t, "1", recorder.Header().Get("RateLimit-Remaining"))
	})

	t.Run("leaves the headers out when the remaining attempts are unknown", func(t *testing.T) {
		t.Parallel()

		a := newRateLimitedAPI(t, nil, errors.New("remaining failed"))
		recorder := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)

		allowed := a.enforceRateLimit(recorder, request, apitest.Faker(t).LetterN(16), 2, time.Minute)

		assert.True(t, allowed)
		assert.Empty(t, recorder.Header().Get("RateLimit-Limit"))
		assert.Empty(t, recorder.Header().Get("RateLimit-Remaining"))
		assert.Empty(t, recorder.Header().Get("RateLimit-Reset"))
	})

	t.Run("answers a failed count with a server error", func(t *testing.T) {
		t.Parallel()

		a := newRateLimitedAPI(t, errors.New("allow failed"), nil)
		recorder := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)

		allowed := a.enforceRateLimit(recorder, request, apitest.Faker(t).LetterN(16), 2, time.Minute)

		assert.False(t, allowed)
		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
		assert.Equal(t, codeServerError, errorCode(t, recorder))
	})
}

// newRateLimitedAPI builds an API whose in-memory limiter fails Allow and Remaining with the given errors.
func newRateLimitedAPI(t *testing.T, allowErr, remainingErr error) *API {
	t.Helper()

	limiter, err := ratelimiter.New()
	require.NoError(t, err)
	return &API{
		logger: zerolog.Nop(),
		rateLimiter: &failingLimiter{
			Limiter:      limiter,
			allowErr:     allowErr,
			remainingErr: remainingErr,
		},
	}
}

// failingLimiter stands in for a limiter whose backend fails.
type failingLimiter struct {
	ratelimiter.Limiter
	allowErr     error
	remainingErr error
}

// Allow fails with allowErr if set and counts the attempt otherwise.
func (l *failingLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, int64, error) {
	if l.allowErr != nil {
		return false, 0, l.allowErr
	}
	return l.Limiter.Allow(ctx, key, limit, window)
}

// Remaining fails with remainingErr if set and returns the remaining attempts otherwise.
func (l *failingLimiter) Remaining(ctx context.Context, key string, limit int) (int, error) {
	if l.remainingErr != nil {
		return 0, l.remainingErr
	}
	return l.Limiter.Remaining(ctx, key, limit)
}
