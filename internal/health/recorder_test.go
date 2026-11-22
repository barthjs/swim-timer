package health

import (
	"errors"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
)

// errUnreachable is the failure the recorder tests report.
var errUnreachable = errors.New("unreachable")

func TestNewRecorder(t *testing.T) {
	t.Parallel()

	recorder := NewRecorder()

	err := recorder.Ping(t.Context())

	assert.NoError(t, err, "a new recorder is healthy")
}

func TestRecorder_RecordSuccess(t *testing.T) {
	t.Parallel()

	t.Run("recovers from a failure and reports the recovery", func(t *testing.T) {
		t.Parallel()

		var changes []bool
		recorder := NewRecorder(WithOnChange(func(healthy bool) { changes = append(changes, healthy) }))
		recorder.RecordFailure(errUnreachable)

		recorder.RecordSuccess()

		assert.Equal(t, []bool{false, true}, changes)
		assert.NoError(t, recorder.Ping(t.Context()))
	})

	t.Run("reports no change for a repeated success", func(t *testing.T) {
		t.Parallel()

		var changes []bool
		recorder := NewRecorder(WithOnChange(func(healthy bool) { changes = append(changes, healthy) }))

		recorder.RecordSuccess()

		assert.Empty(t, changes)
	})
}

func TestRecorder_RecordFailure(t *testing.T) {
	t.Parallel()

	t.Run("marks the dependency as unreachable and reports it once", func(t *testing.T) {
		t.Parallel()

		var changes []bool
		recorder := NewRecorder(WithOnChange(func(healthy bool) { changes = append(changes, healthy) }))

		recorder.RecordFailure(errUnreachable)
		recorder.RecordFailure(errUnreachable)

		assert.Equal(t, []bool{false}, changes)
		assert.ErrorIs(t, recorder.Ping(t.Context()), errUnreachable)
	})

	t.Run("reports the expired failure as healthy", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) {
			var mu sync.Mutex
			var changes []bool
			recorder := NewRecorder(WithOnChange(func(healthy bool) {
				mu.Lock()
				defer mu.Unlock()
				changes = append(changes, healthy)
			}))

			recorder.RecordFailure(errUnreachable)
			time.Sleep(failureTTL + time.Second)
			synctest.Wait()

			mu.Lock()
			defer mu.Unlock()
			assert.Equal(t, []bool{false, true}, changes)
			assert.NoError(t, recorder.Ping(t.Context()))
		})
	})
}

func TestRecorder_Ping(t *testing.T) {
	t.Parallel()

	t.Run("reports an expired failure as healthy", func(t *testing.T) {
		t.Parallel()

		recorder := NewRecorder()
		recorder.RecordFailure(errUnreachable)
		// Backdate the failure instead of waiting for the TTL
		recorder.recorded = time.Now().Add(-failureTTL - time.Second)

		err := recorder.Ping(t.Context())

		assert.NoError(t, err)
		assert.ErrorIs(t, recorder.err, errUnreachable, "only expire clears the failure, since it reports the change")
	})

	t.Run("is safe for concurrent use", func(t *testing.T) {
		t.Parallel()

		recorder := NewRecorder()
		var wg sync.WaitGroup

		for i := range 50 {
			wg.Go(func() {
				if i%2 == 0 {
					recorder.RecordFailure(errUnreachable)
				} else {
					recorder.RecordSuccess()
				}
				_ = recorder.Ping(t.Context())
			})
		}
		wg.Wait()
	})
}
