package ratelimiter_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/ratelimiter"
)

func TestMemoryLimiter(t *testing.T) {
	t.Parallel()

	runLimiterContract(t, newLimiter(t))
}

func TestMemoryLimiter_Allow(t *testing.T) {
	t.Parallel()

	t.Run("allows the attempts within the limit", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		limiter := newLimiter(t)

		for i := range 3 {
			allowed, _, err := limiter.Allow(ctx, "client", 3, time.Minute)

			require.NoError(t, err)
			assert.True(t, allowed, "attempt %d", i+1)
		}
	})

	t.Run("denies attempts beyond the limit with a retry time", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		clock := newFakeClock()
		limiter := newLimiter(t, ratelimiter.WithClock(clock.Now))
		for range 3 {
			_, _, err := limiter.Allow(ctx, "client", 3, time.Minute)
			require.NoError(t, err)
		}
		clock.Advance(20 * time.Second)

		allowed, retryAfter, err := limiter.Allow(ctx, "client", 3, time.Minute)

		require.NoError(t, err)
		assert.False(t, allowed)
		assert.Equal(t, int64(40), retryAfter)
	})

	t.Run("counts keys separately", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		limiter := newLimiter(t)
		for range 5 {
			_, _, _ = limiter.Allow(ctx, "first", 2, time.Minute)
		}

		allowed, _, err := limiter.Allow(ctx, "second", 2, time.Minute)

		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("starts a new window after expiry", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		clock := newFakeClock()
		limiter := newLimiter(t, ratelimiter.WithClock(clock.Now))
		for range 5 {
			_, _, _ = limiter.Allow(ctx, "client", 2, time.Minute)
		}
		clock.Advance(time.Minute + time.Second)

		allowed, retryAfter, err := limiter.Allow(ctx, "client", 2, time.Minute)

		require.NoError(t, err)
		assert.True(t, allowed)
		assert.Equal(t, int64(0), retryAfter)
	})

	t.Run("keeps the window of the first attempt", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		clock := newFakeClock()
		limiter := newLimiter(t, ratelimiter.WithClock(clock.Now))
		_, _, _ = limiter.Allow(ctx, "client", 5, time.Minute)
		clock.Advance(50 * time.Second)

		_, _, _ = limiter.Allow(ctx, "client", 5, time.Minute)

		assert.Equal(t, int64(10), limiter.AvailableIn(ctx, "client"))
	})

	t.Run("ignores an attempt in a closed window", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		clock := newFakeClock()
		limiter := newLimiter(t, ratelimiter.WithClock(clock.Now))
		require.NoError(t, limiter.Hit(ctx, "login", time.Minute))
		clock.Advance(time.Minute)

		allowed, _, err := limiter.Allow(ctx, "login", 1, time.Minute)

		require.NoError(t, err)
		assert.True(t, allowed, "the closed window does not count the earlier hit")
	})

	t.Run("is safe for concurrent use", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		limiter := newLimiter(t)
		var allowedCount atomic.Int32
		var wg sync.WaitGroup

		for range 100 {
			wg.Go(func() {
				if allowed, _, _ := limiter.Allow(ctx, "client", 10, time.Minute); allowed {
					allowedCount.Add(1)
				}
			})
		}
		wg.Wait()

		remaining, err := limiter.Remaining(ctx, "client", 10)
		require.NoError(t, err)
		assert.Equal(t, 0, remaining)
		assert.LessOrEqual(t, allowedCount.Load(), int32(10))
	})
}

func TestMemoryLimiter_TooManyAttempts(t *testing.T) {
	t.Parallel()

	t.Run("reports too many attempts at the limit", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		limiter := newLimiter(t, ratelimiter.WithClock(newFakeClock().Now))
		require.NoError(t, limiter.Hit(ctx, "login", time.Minute))
		require.NoError(t, limiter.Hit(ctx, "login", time.Minute))

		tooMany, err := limiter.TooManyAttempts(ctx, "login", 2)

		require.NoError(t, err)
		assert.True(t, tooMany)
	})

	t.Run("keeps a key below the limit allowed", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		limiter := newLimiter(t, ratelimiter.WithClock(newFakeClock().Now))
		require.NoError(t, limiter.Hit(ctx, "login", time.Minute))

		tooMany, err := limiter.TooManyAttempts(ctx, "login", 2)

		require.NoError(t, err)
		assert.False(t, tooMany)
	})

	t.Run("keeps an unknown key allowed", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		limiter := newLimiter(t, ratelimiter.WithClock(newFakeClock().Now))

		tooMany, err := limiter.TooManyAttempts(ctx, "login", 2)

		require.NoError(t, err)
		assert.False(t, tooMany, "an unknown key has no attempts")
	})

	t.Run("closes the window at its expiry", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		clock := newFakeClock()
		limiter := newLimiter(t, ratelimiter.WithClock(clock.Now))
		require.NoError(t, limiter.Hit(ctx, "login", time.Minute))
		clock.Advance(time.Minute)

		tooMany, err := limiter.TooManyAttempts(ctx, "login", 1)

		require.NoError(t, err)
		assert.False(t, tooMany)
	})

	t.Run("ignores expired attempts", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		clock := newFakeClock()
		limiter := newLimiter(t, ratelimiter.WithClock(clock.Now))
		require.NoError(t, limiter.Hit(ctx, "login", time.Minute))
		require.NoError(t, limiter.Hit(ctx, "login", time.Minute))
		clock.Advance(time.Minute + time.Second)

		tooMany, err := limiter.TooManyAttempts(ctx, "login", 2)

		require.NoError(t, err)
		assert.False(t, tooMany)
	})
}

func TestMemoryLimiter_Hit(t *testing.T) {
	t.Parallel()

	t.Run("restarts an expired window", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		clock := newFakeClock()
		limiter := newLimiter(t, ratelimiter.WithClock(clock.Now))
		require.NoError(t, limiter.Hit(ctx, "login", time.Minute))
		require.NoError(t, limiter.Hit(ctx, "login", time.Minute))
		clock.Advance(time.Minute + time.Second)

		require.NoError(t, limiter.Hit(ctx, "login", time.Minute))

		remaining, err := limiter.Remaining(ctx, "login", 5)
		require.NoError(t, err)
		assert.Equal(t, 4, remaining)
	})

	t.Run("extends the window", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		clock := newFakeClock()
		limiter := newLimiter(t, ratelimiter.WithClock(clock.Now))
		require.NoError(t, limiter.Hit(ctx, "login", time.Second))

		require.NoError(t, limiter.Hit(ctx, "login", time.Hour))

		assert.Equal(t, int64(3600), limiter.AvailableIn(ctx, "login"))
	})
}

func TestMemoryLimiter_Remaining(t *testing.T) {
	t.Parallel()

	t.Run("returns the full budget for an unknown key", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		limiter := newLimiter(t)

		remaining, err := limiter.Remaining(ctx, "client", 3)

		require.NoError(t, err)
		assert.Equal(t, 3, remaining)
	})

	t.Run("counts down with every hit", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		limiter := newLimiter(t)
		require.NoError(t, limiter.Hit(ctx, "client", time.Minute))

		remaining, err := limiter.Remaining(ctx, "client", 3)

		require.NoError(t, err)
		assert.Equal(t, 2, remaining)
	})

	t.Run("keeps zero beyond the limit", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		limiter := newLimiter(t)
		for range 6 {
			require.NoError(t, limiter.Hit(ctx, "client", time.Minute))
		}

		remaining, err := limiter.Remaining(ctx, "client", 3)

		require.NoError(t, err)
		assert.Equal(t, 0, remaining, "never negative")
	})

	t.Run("ignores expired attempts", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		clock := newFakeClock()
		limiter := newLimiter(t, ratelimiter.WithClock(clock.Now))
		require.NoError(t, limiter.Hit(ctx, "login", time.Minute))
		require.NoError(t, limiter.Hit(ctx, "login", time.Minute))
		clock.Advance(time.Minute + time.Second)

		remaining, err := limiter.Remaining(ctx, "login", 2)

		require.NoError(t, err)
		assert.Equal(t, 2, remaining)
	})
}

func TestMemoryLimiter_AvailableIn(t *testing.T) {
	t.Parallel()

	t.Run("returns the seconds until the window closes", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		limiter := newLimiter(t, ratelimiter.WithClock(newFakeClock().Now))
		require.NoError(t, limiter.Hit(ctx, "login", time.Minute))
		require.NoError(t, limiter.Hit(ctx, "login", time.Minute))

		availableIn := limiter.AvailableIn(ctx, "login")

		assert.Equal(t, int64(60), availableIn)
	})

	t.Run("rounds up to whole seconds", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		clock := newFakeClock()
		limiter := newLimiter(t, ratelimiter.WithClock(clock.Now))
		require.NoError(t, limiter.Hit(ctx, "login", time.Minute))
		clock.Advance(59*time.Second + 500*time.Millisecond)

		availableIn := limiter.AvailableIn(ctx, "login")

		assert.Equal(t, int64(1), availableIn)
	})

	t.Run("returns zero for an unknown key", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		limiter := newLimiter(t)

		availableIn := limiter.AvailableIn(ctx, "unknown")

		assert.Equal(t, int64(0), availableIn)
	})

	t.Run("returns zero at the expiry of the window", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		clock := newFakeClock()
		limiter := newLimiter(t, ratelimiter.WithClock(clock.Now))
		require.NoError(t, limiter.Hit(ctx, "login", time.Minute))
		clock.Advance(time.Minute)

		availableIn := limiter.AvailableIn(ctx, "login")

		assert.Equal(t, int64(0), availableIn)
	})

	t.Run("returns zero after the expiry of the window", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		clock := newFakeClock()
		limiter := newLimiter(t, ratelimiter.WithClock(clock.Now))
		require.NoError(t, limiter.Hit(ctx, "login", time.Minute))
		require.NoError(t, limiter.Hit(ctx, "login", time.Minute))
		clock.Advance(time.Minute + time.Second)

		availableIn := limiter.AvailableIn(ctx, "login")

		assert.Equal(t, int64(0), availableIn)
	})
}

func TestMemoryLimiter_Reset(t *testing.T) {
	t.Parallel()

	t.Run("removes the attempts", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		limiter := newLimiter(t, ratelimiter.WithClock(newFakeClock().Now))
		require.NoError(t, limiter.Hit(ctx, "login", time.Minute))
		require.NoError(t, limiter.Hit(ctx, "login", time.Minute))

		require.NoError(t, limiter.Reset(ctx, "login"))

		tooMany, err := limiter.TooManyAttempts(ctx, "login", 2)
		require.NoError(t, err)
		assert.False(t, tooMany, "a reset clears the attempts")
	})
}

func TestMemoryLimiter_ClearExpired(t *testing.T) {
	t.Parallel()

	t.Run("removes the expired entries and keeps the active ones", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		clock := newFakeClock()
		limiter := newLimiter(t, ratelimiter.WithClock(clock.Now))
		require.NoError(t, limiter.Hit(ctx, "expired", time.Minute))
		clock.Advance(30 * time.Second)
		require.NoError(t, limiter.Hit(ctx, "active", time.Minute))
		clock.Advance(45 * time.Second)
		prunable, ok := limiter.(ratelimiter.Prunable)
		require.True(t, ok)

		require.NoError(t, prunable.ClearExpired(ctx))

		// Moving the clock back revives the windows of the entries that were kept,
		// so a cleared entry shows up as a full budget.
		clock.Advance(-60 * time.Second)
		remaining, err := limiter.Remaining(ctx, "expired", 3)
		require.NoError(t, err)
		assert.Equal(t, 3, remaining, "the expired entry was removed")
		remaining, err = limiter.Remaining(ctx, "active", 3)
		require.NoError(t, err)
		assert.Equal(t, 2, remaining, "the active entry was kept")
	})
}

// fakeClock is a time source the tests move forward by hand instead of waiting.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

// newFakeClock creates a fake clock set to a fixed time.
func newFakeClock() *fakeClock {
	return &fakeClock{
		now: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
	}
}

// Now returns the current fake time.
func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Advance moves the fake time forward by d.
func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}
