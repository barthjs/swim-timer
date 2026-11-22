package server

import (
	"context"
	"io"
	"net/http"
	"time"
)

// limitedBody is a request body limited by BodyLimit. It keeps the original body,
// so a later BodyLimit for a route replaces the limit of a group instead of narrowing it.
type limitedBody struct {
	io.ReadCloser
	original io.ReadCloser
}

// BodyLimit limits the request body to maxBytes, or removes the limit for 0, and replaces a limit applied before,
// so a group or a route can change the default.
func BodyLimit(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			original := r.Body
			if limited, ok := original.(*limitedBody); ok {
				original = limited.original
			}

			if maxBytes <= 0 {
				r.Body = original
			} else if original != nil && original != http.NoBody {
				r.Body = &limitedBody{
					ReadCloser: http.MaxBytesReader(w, original, maxBytes),
					original:   original,
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Timeout sets the deadlines of the connection and the request context, or removes them for 0,
// and replaces a timeout applied before, so a group or a route can change the default.
func Timeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			controller := http.NewResponseController(w)

			// The zero deadline lets long-lived responses run
			var deadline time.Time
			if timeout > 0 {
				deadline = time.Now().Add(timeout)
			}
			// Writers that do not support deadlines, such as test recorders, keep only the context deadline
			_ = controller.SetReadDeadline(deadline)
			_ = controller.SetWriteDeadline(deadline)

			// The deadline derives from the request context before any Timeout ran,
			// so a longer timeout for a route is not cut short by a shorter one of its group. Cancellation by the client still applies.
			base, ok := r.Context().Value(timeoutBaseKey{}).(context.Context)
			if !ok {
				base = r.Context()
			}
			var ctx context.Context = detachedContext{
				Context: context.WithoutCancel(r.Context()),
				base:    base,
			}
			ctx = context.WithValue(ctx, timeoutBaseKey{}, base)

			if timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithDeadline(ctx, deadline)
				defer cancel()
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// timeoutBaseKey is the context key of the context a timeout is detached from.
type timeoutBaseKey struct{}

// detachedContext carries the values of the current request context but the deadline and cancellation of base.
type detachedContext struct {
	context.Context
	base context.Context
}

// Deadline returns the deadline of the base context.
func (c detachedContext) Deadline() (time.Time, bool) { return c.base.Deadline() }

// Done returns the done channel of the base context.
func (c detachedContext) Done() <-chan struct{} { return c.base.Done() }

// Err returns the error of the base context.
func (c detachedContext) Err() error { return c.base.Err() }
