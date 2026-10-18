package apitest_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/barthjs/swim-timer/internal/apitest"
)

// TestFaker does not run in parallel, since its subtests set the seed in the environment.
func TestFaker(t *testing.T) {
	t.Run("reproduces the seed", func(t *testing.T) {
		t.Setenv(apitest.SeedEnv, "42")
		first := apitest.Faker(t).Name()

		second := apitest.Faker(t).Name()

		assert.Equal(t, first, second)
	})

	t.Run("logs the seed once the test failed", func(t *testing.T) {
		t.Setenv(apitest.SeedEnv, "7")
		mock := &loggingT{
			TB: t,
		}
		apitest.Faker(mock)
		mock.failed = true

		mock.runCleanups()

		assert.Contains(t, mock.logged, "APITEST_SEED=7")
	})

	t.Run("rejects an invalid seed", func(t *testing.T) {
		t.Setenv(apitest.SeedEnv, "not a number")
		mock := &recordingT{
			TB: t,
		}

		create := func() { apitest.Faker(mock) }

		assert.Panics(t, create)
		assert.True(t, mock.failed)
	})
}

// loggingT collects log output and cleanups, to check what the faker logs once a test failed.
type loggingT struct {
	recordingT
	logged   string
	cleanups []func()
}

// Logf records the formatted message.
func (l *loggingT) Logf(format string, args ...any) { l.logged += fmt.Sprintf(format, args...) }

// Cleanup registers f to run when runCleanups is called.
func (l *loggingT) Cleanup(f func()) { l.cleanups = append(l.cleanups, f) }

// Failed reports whether the test is marked as failed.
func (l *loggingT) Failed() bool { return l.failed }

// runCleanups runs the registered cleanup functions in reverse order.
func (l *loggingT) runCleanups() {
	for _, v := range slices.Backward(l.cleanups) {
		v()
	}
}
