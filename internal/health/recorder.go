package health

import (
	"context"
	"sync"
	"time"
)

// failureTTL bounds how long a single recorded failure keeps a Recorder unhealthy. A Recorder only ever observes outcomes
// its caller happens to report, never probing the dependency itself, so a failure that is never followed
// by a fresh attempt would otherwise wedge it as unavailable forever.
const failureTTL = 60 * time.Second

// Recorder is a Pinger for dependencies that cannot be probed synchronously on demand. The caller records the outcome
// of its own operations, and Ping reports the most recently recorded one until it expires.
type Recorder struct {
	mu       sync.Mutex
	err      error
	recorded time.Time
}

// NewRecorder creates a Recorder that starts healthy.
func NewRecorder() *Recorder {
	return &Recorder{}
}

// RecordSuccess marks the dependency as reachable.
func (r *Recorder) RecordSuccess() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.err = nil
}

// RecordFailure marks the dependency as unreachable, with err as the reason, for at most failureTTL.
func (r *Recorder) RecordFailure(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.err = err
	r.recorded = time.Now()
}

// Ping returns the error from the most recent RecordFailure call, or nil if the last recorded outcome was a success,
// none was recorded yet, or the failure has expired.
func (r *Recorder) Ping(_ context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.err != nil && time.Since(r.recorded) > failureTTL {
		r.err = nil
	}
	return r.err
}
