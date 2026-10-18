package api

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPI_Background(t *testing.T) {
	t.Parallel()

	t.Run("ends with the application", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(t.Context())
		wg := &sync.WaitGroup{}
		a := New(ctx, Deps{
			Logger: zerolog.Nop(),
			WG:     wg,
		})

		started := make(chan struct{})
		ended := make(chan error, 1)
		a.background(time.Hour, func(ctx context.Context) {
			close(started)
			<-ctx.Done()
			ended <- ctx.Err()
		})
		<-started
		cancel()

		waitForGroup(t, wg)
		assert.ErrorIs(t, <-ended, context.Canceled, "the work ends with the application, not after its timeout")
	})

	t.Run("ends after the timeout", func(t *testing.T) {
		t.Parallel()
		wg := &sync.WaitGroup{}
		a := New(t.Context(), Deps{
			Logger: zerolog.Nop(),
			WG:     wg,
		})

		ended := make(chan error, 1)
		a.background(time.Millisecond, func(ctx context.Context) {
			<-ctx.Done()
			ended <- ctx.Err()
		})

		waitForGroup(t, wg)
		assert.ErrorIs(t, <-ended, context.DeadlineExceeded)
	})

	t.Run("recovers panics", func(t *testing.T) {
		t.Parallel()
		wg := &sync.WaitGroup{}
		a := New(t.Context(), Deps{
			Logger: zerolog.Nop(),
			WG:     wg,
		})

		a.background(time.Minute, func(context.Context) { panic("background work failed") })

		waitForGroup(t, wg)
	})
}

// waitForGroup fails the test if the work counted in wg does not end in time.
func waitForGroup(t *testing.T, wg *sync.WaitGroup) {
	t.Helper()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the background work did not end")
	}
}
