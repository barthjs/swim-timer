// Package ratelimiter provides fixed-window rate limiting.
package ratelimiter

import (
	"context"
	"time"
)

// Limiter counts attempts per key within a fixed time window.
// Allow lets exactly limit attempts through per window and rejects the ones after them.
//
// Two usage patterns are supported:
//   - Allow counts every attempt, for plain request limits.
//   - TooManyAttempts + Hit + Reset count only real failures:
//     check the budget first, Hit on a failed attempt, Reset on success.
type Limiter interface {
	// Allow atomically counts an attempt for a key and reports whether it is still
	// within the limit for the current window, plus the seconds until reset when it is not.
	Allow(ctx context.Context, key string, limit int, window time.Duration) (allowed bool, retryAfter int64, err error)
	// TooManyAttempts reports whether a key has already reached the limit, without counting a hit.
	TooManyAttempts(ctx context.Context, key string, limit int) (bool, error)
	// Hit counts a single attempt for a key and (re)sets its window.
	Hit(ctx context.Context, key string, window time.Duration) error
	// Reset removes all recorded attempts for a key.
	Reset(ctx context.Context, key string) error

	// Remaining returns the number of allowed attempts left in the current window.
	// It returns 0 if the limit has been reached or exceeded.
	Remaining(ctx context.Context, key string, limit int) (int, error)
	// AvailableIn returns the seconds until the key's window resets, or 0 when unknown.
	AvailableIn(ctx context.Context, key string) (retryAfter int64)
}

// Prunable is implemented by limiters whose expired entries must be removed by a periodic job.
type Prunable interface {
	// ClearExpired removes all entries that have expired.
	ClearExpired(ctx context.Context) error
}

// Option configures a Limiter.
type Option func(*options)

// options holds the settings applied by the Option values.
type options struct {
	now func() time.Time
}

// WithClock sets the time source the windows are measured with, for example a fake clock in tests.
func WithClock(now func() time.Time) Option {
	return func(o *options) {
		if now != nil {
			o.now = now
		}
	}
}

// New creates a Limiter that keeps its counters in process memory.
func New(opts ...Option) (Limiter, error) {
	o := options{
		now: time.Now,
	}
	for _, opt := range opts {
		opt(&o)
	}
	return newMemoryLimiter(o.now), nil
}

// RunCleanup clears expired keys at an interval and returns once ctx is canceled.
// For a limiter without expiring keys, it returns right away.
func RunCleanup(ctx context.Context, limiter Limiter, interval time.Duration) {
	prunable, ok := limiter.(Prunable)
	if !ok {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = prunable.ClearExpired(ctx)
		}
	}
}
