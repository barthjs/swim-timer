package apitest_test

import (
	"testing"
)

// recordingT records failures instead of failing the real test, to check that an assertion fails.
// FailNow panics, since stopping the goroutine would end the real test.
type recordingT struct {
	testing.TB
	failed bool
}

// Errorf marks the test as failed instead of reporting the message.
func (r *recordingT) Errorf(string, ...any) { r.failed = true }

// Fail marks the test as failed.
func (r *recordingT) Fail() { r.failed = true }

// FailNow marks the test as failed and stops the caller with a panic.
func (r *recordingT) FailNow() { r.failed = true; panic("FailNow") }

// Helper does nothing.
func (r *recordingT) Helper() {}
