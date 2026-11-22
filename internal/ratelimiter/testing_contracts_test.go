package ratelimiter_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/ratelimiter"
)

// runLimiterContract checks the behavior every Limiter backend shares, in sequential subtests with keys of their own,
// since a database backed limiter shares one table.
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
