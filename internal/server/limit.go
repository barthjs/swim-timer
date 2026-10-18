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

// BodyLimit limits the request body to maxBytes. Reading beyond it fails with *http.MaxBytesError.
// Applied globally it sets the default, and applied again to a group or a route,
// it replaces the default for that route. A limit of 0 or less removes the limit.
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

// Timeout bounds the time a route has to read the request and write the response.
// It sets the read and write deadline of the connection, which replaces the server-wide timeouts for this request,
// and a context deadline handlers can react to. Applied again to a group or a route, it replaces the earlier timeout.
// A timeout of 0 or fewer removes both, for long-lived responses such as streams.
func Timeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			controller := http.NewResponseController(w)

			var deadline time.Time
			if timeout > 0 {
				deadline = time.Now().Add(timeout)
			}
			// Writers that do not support deadlines, such as test recorders, keep only the context deadline
			_ = controller.SetReadDeadline(deadline)
			_ = controller.SetWriteDeadline(deadline)

			// The deadline derives from the request context before any Timeout ran, so a longer timeout for a route
			// is not cut short by a shorter one of its group. Cancellation by the client still applies.
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
