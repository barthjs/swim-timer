package ratelimiter_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/ratelimiter"
)

func TestWithClock(t *testing.T) {
	t.Parallel()

	t.Run("ignores a nil clock", func(t *testing.T) {
		t.Parallel()

		limiter := newLimiter(t, ratelimiter.WithClock(nil))

		allowed, _, err := limiter.Allow(t.Context(), "client", 1, time.Minute)

		require.NoError(t, err)
		assert.True(t, allowed)
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("returns a prunable memory limiter", func(t *testing.T) {
		t.Parallel()

		limiter := newLimiter(t)

		_, ok := limiter.(ratelimiter.Prunable)
		assert.True(t, ok, "the in-memory limiter needs the periodic cleanup")
	})
}

func TestRunCleanup(t *testing.T) {
	t.Parallel()

	t.Run("clears expired entries periodically until canceled", func(t *testing.T) {
		t.Parallel()

		limiter := &countingLimiter{
			Limiter: newLimiter(t),
			cleared: make(chan struct{}, 1),
		}
		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan struct{})
		go func() {
			ratelimiter.RunCleanup(ctx, limiter, time.Millisecond)
			close(done)
		}()

		select {
		case <-limiter.cleared:
		case <-time.After(time.Second):
			require.FailNow(t, "ClearExpired was not called")
		}

		cancel()
		select {
		case <-done:
		case <-time.After(time.Second):
			require.FailNow(t, "RunCleanup did not return after cancel")
		}
	})

	t.Run("returns right away for limiters without expiry", func(t *testing.T) {
		t.Parallel()

		var limiter struct{ ratelimiter.Limiter }
		ratelimiter.RunCleanup(t.Context(), limiter, time.Millisecond)
	})
}

// countingLimiter records how often ClearExpired runs.
type countingLimiter struct {
	ratelimiter.Limiter
	cleared chan struct{}
}

// ClearExpired signals the call on the cleared channel without blocking.
func (c *countingLimiter) ClearExpired(context.Context) error {
	select {
	case c.cleared <- struct{}{}:
	default:
	}
	return nil
}

// runLimiterContract checks the behavior every Limiter backend must share, so the backends stay interchangeable.
// The subtests run sequentially, since a database backed limiter shares one table. Every subtest uses its own key,
// so they do not see each other's attempts.
func runLimiterContract(t *testing.T, limiter ratelimiter.Limiter) {
	t.Helper()
	ctx := t.Context()
	window := time.Minute

	t.Run("allow lets exactly limit attempts through", func(t *testing.T) {
		key := "contract-allow"
		require.NoError(t, limiter.Reset(ctx, key))

		for i := range 3 {
			allowed, retryAfter, err := limiter.Allow(ctx, key, 3, window)
			require.NoError(t, err)
			assert.True(t, allowed, "attempt %d", i+1)
			assert.Equal(t, int64(0), retryAfter)
		}

		allowed, retryAfter, err := limiter.Allow(ctx, key, 3, window)
		require.NoError(t, err)
		assert.False(t, allowed)
		assert.Positive(t, retryAfter)
		assert.LessOrEqual(t, retryAfter, int64(window/time.Second))
	})

	t.Run("hit counts failures until too many attempts", func(t *testing.T) {
		key := "contract-hit"
		require.NoError(t, limiter.Reset(ctx, key))

		tooMany, err := limiter.TooManyAttempts(ctx, key, 2)
		require.NoError(t, err)
		assert.False(t, tooMany)

		require.NoError(t, limiter.Hit(ctx, key, window))
		tooMany, err = limiter.TooManyAttempts(ctx, key, 2)
		require.NoError(t, err)
		assert.False(t, tooMany)

		require.NoError(t, limiter.Hit(ctx, key, window))
		tooMany, err = limiter.TooManyAttempts(ctx, key, 2)
		require.NoError(t, err)
		assert.True(t, tooMany)
		assert.Positive(t, limiter.AvailableIn(ctx, key))
	})

	t.Run("remaining counts down to zero", func(t *testing.T) {
		key := "contract-remaining"
		require.NoError(t, limiter.Reset(ctx, key))

		remaining, err := limiter.Remaining(ctx, key, 2)
		require.NoError(t, err)
		assert.Equal(t, 2, remaining)

		for range 3 {
			require.NoError(t, limiter.Hit(ctx, key, window))
		}
		remaining, err = limiter.Remaining(ctx, key, 2)
		require.NoError(t, err)
		assert.Equal(t, 0, remaining)
	})

	t.Run("reset forgets all attempts", func(t *testing.T) {
		key := "contract-reset"
		require.NoError(t, limiter.Hit(ctx, key, window))
		require.NoError(t, limiter.Reset(ctx, key))

		remaining, err := limiter.Remaining(ctx, key, 5)
		require.NoError(t, err)
		assert.Equal(t, 5, remaining)
		assert.Equal(t, int64(0), limiter.AvailableIn(ctx, key))
	})
}

// newLimiter creates an in-memory limiter for a test.
func newLimiter(t *testing.T, opts ...ratelimiter.Option) ratelimiter.Limiter {
	t.Helper()

	limiter, err := ratelimiter.New(opts...)
	require.NoError(t, err)
	return limiter
}
