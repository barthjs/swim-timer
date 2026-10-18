package health

import (
	"errors"
	"sync"
	"testing"
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

	recorder := NewRecorder()
	recorder.RecordFailure(errUnreachable)

	recorder.RecordSuccess()

	assert.NoError(t, recorder.Ping(t.Context()), "a success recovers from a failure")
}

func TestRecorder_RecordFailure(t *testing.T) {
	t.Parallel()

	recorder := NewRecorder()

	recorder.RecordFailure(errUnreachable)

	assert.ErrorIs(t, recorder.Ping(t.Context()), errUnreachable)
}

func TestRecorder_Ping(t *testing.T) {
	t.Parallel()

	t.Run("forgets an expired failure", func(t *testing.T) {
		t.Parallel()

		recorder := NewRecorder()
		recorder.RecordFailure(errUnreachable)
		// Backdate the failure instead of waiting for the TTL
		recorder.recorded = time.Now().Add(-failureTTL - time.Second)

		err := recorder.Ping(t.Context())

		assert.NoError(t, err)
		assert.NoError(t, recorder.err, "the expired failure is cleared")
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
