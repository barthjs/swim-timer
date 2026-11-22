package health

import (
	"context"
	"sync"
	"time"
)

// failureTTL bounds how long a single recorded failure keeps a Recorder unhealthy.
// A Recorder only ever observes outcomes its caller happens to report, never probing the dependency itself,
// so a failure that is never followed by a fresh attempt would otherwise wedge it as unavailable forever.
const failureTTL = 60 * time.Second

// Recorder is a Pinger for dependencies that cannot be probed synchronously on demand.
// The caller records the outcome of its own operations, and Ping reports the most recently recorded one until it expires.
type Recorder struct {
	mu       sync.Mutex
	err      error
	recorded time.Time
	onChange func(healthy bool)
	expiry   *time.Timer
}

// RecorderOption configures a Recorder.
type RecorderOption func(*Recorder)

// WithOnChange calls onChange whenever the dependency turns unhealthy or healthy again, also when a failure expires.
func WithOnChange(onChange func(healthy bool)) RecorderOption {
	return func(r *Recorder) {
		r.onChange = onChange
	}
}

// NewRecorder creates a Recorder that starts healthy.
func NewRecorder(opts ...RecorderOption) *Recorder {
	r := &Recorder{}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// RecordSuccess marks the dependency as reachable.
func (r *Recorder) RecordSuccess() {
	r.record(nil)
}

// RecordFailure marks the dependency as unreachable, with err as the reason, for at most failureTTL.
func (r *Recorder) RecordFailure(err error) {
	r.record(err)
}

// Ping returns the error from the most recent RecordFailure call, or nil if the last recorded outcome was a success,
// none was recorded yet, or the failure has expired.
func (r *Recorder) Ping(_ context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// An expired failure is only cleared by expire, which also reports the change
	if r.err != nil && time.Since(r.recorded) > failureTTL {
		return nil
	}
	return r.err
}

// record stores an outcome and reports a change of the health. A failure arms the timer that reports its expiry.
func (r *Recorder) record(err error) {
	r.mu.Lock()
	wasHealthy := r.err == nil
	r.err = err
	r.recorded = time.Now()
	if r.expiry != nil {
		r.expiry.Stop()
		r.expiry = nil
	}
	if err != nil && r.onChange != nil {
		r.expiry = time.AfterFunc(failureTTL, r.expire)
	}
	onChange := r.onChange
	r.mu.Unlock()

	if onChange != nil && wasHealthy != (err == nil) {
		onChange(err == nil)
	}
}

// expire clears a failure that outlived failureTTL and reports the dependency as healthy again.
func (r *Recorder) expire() {
	r.mu.Lock()
	if r.err == nil || time.Since(r.recorded) < failureTTL {
		r.mu.Unlock()
		return
	}
	r.err = nil
	r.expiry = nil
	onChange := r.onChange
	r.mu.Unlock()

	onChange(true)
}
