package ratelimiter_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/ratelimiter"
)

// newLimiter creates an in-memory limiter for a test.
func newLimiter(t *testing.T, opts ...ratelimiter.Option) ratelimiter.Limiter {
	t.Helper()

	limiter, err := ratelimiter.New(opts...)
	require.NoError(t, err)
	return limiter
}
