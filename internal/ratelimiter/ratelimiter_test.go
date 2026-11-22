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
