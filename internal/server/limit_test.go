package server_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/server"
)

func TestBodyLimit(t *testing.T) {
	t.Parallel()

	t.Run("accepts a request without a body", func(t *testing.T) {
		t.Parallel()

		handler := server.BodyLimit(10)(http.HandlerFunc(readAll))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	tests := []struct {
		name       string
		middleware []func(http.Handler) http.Handler
		size       int
		status     int
	}{
		{
			name:       "accepts a body within the limit",
			middleware: []func(http.Handler) http.Handler{server.BodyLimit(10)},
			size:       10,
			status:     http.StatusOK,
		},
		{
			name:       "lets a route raise the default",
			middleware: []func(http.Handler) http.Handler{server.BodyLimit(10), server.BodyLimit(100)},
			size:       50,
			status:     http.StatusOK,
		},
		{
			name:       "lets a route lower the default",
			middleware: []func(http.Handler) http.Handler{server.BodyLimit(100), server.BodyLimit(10)},
			size:       50,
			status:     http.StatusRequestEntityTooLarge,
		},
		{
			name:       "removes the limit with zero",
			middleware: []func(http.Handler) http.Handler{server.BodyLimit(10), server.BodyLimit(0)},
			size:       50,
			status:     http.StatusOK,
		},
		{
			name:       "rejects a body beyond the limit",
			middleware: []func(http.Handler) http.Handler{server.BodyLimit(10)},
			size:       11,
			status:     http.StatusRequestEntityTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var handler http.Handler = http.HandlerFunc(readAll)
			for _, v := range slices.Backward(tt.middleware) {
				handler = v(handler)
			}
			rec := httptest.NewRecorder()

			body := strings.NewReader(strings.Repeat("a", tt.size))
			handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", body))

			assert.Equal(t, tt.status, rec.Code)
		})
	}
}

// readAll answers with the number of bytes read, or 413 if the body was too large.
func readAll(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		return
	}
	_, _ = io.WriteString(w, strings.Repeat("x", len(data)))
}

func TestTimeout(t *testing.T) {
	t.Parallel()

	t.Run("sets a context deadline", func(t *testing.T) {
		t.Parallel()

		deadline, ok := deadlineOf(t, server.Timeout(time.Minute))

		require.True(t, ok)
		assert.WithinDuration(t, time.Now().Add(time.Minute), deadline, 5*time.Second)
	})

	t.Run("sets the connection deadlines", func(t *testing.T) {
		t.Parallel()

		slow := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(200 * time.Millisecond)
			_, _ = io.WriteString(w, "late")
		})
		srv := httptest.NewTestServer(t, server.Timeout(50*time.Millisecond)(slow))

		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL, nil)
		require.NoError(t, err)

		resp, err := srv.Client().Do(request)
		if err == nil {
			defer func() { _ = resp.Body.Close() }()
			_, err = io.ReadAll(resp.Body)
		}

		assert.Error(t, err, "the write deadline cuts the response off")
	})

	t.Run("replaces a shorter group timeout with the timeout of the route", func(t *testing.T) {
		t.Parallel()

		deadline, ok := deadlineOf(t, server.Timeout(time.Second), server.Timeout(time.Hour))

		require.True(t, ok)
		assert.WithinDuration(t, time.Now().Add(time.Hour), deadline, 5*time.Second)
	})

	t.Run("removes the deadline with zero", func(t *testing.T) {
		t.Parallel()

		_, ok := deadlineOf(t, server.Timeout(time.Second), server.Timeout(0))

		assert.False(t, ok)
	})

	t.Run("cancels the context at the deadline", func(t *testing.T) {
		t.Parallel()

		var err error
		handler := server.Timeout(10 * time.Millisecond)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
			err = r.Context().Err()
		}))

		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))

		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("cancels the context with the request", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		var err error
		inner := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			cancel()
			<-r.Context().Done()
			err = r.Context().Err()
		})
		handler := server.Timeout(time.Hour)(server.Timeout(time.Hour)(inner))

		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil))

		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("keeps the values of the request context", func(t *testing.T) {
		t.Parallel()

		var id string
		inner := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			id = server.RequestIDFromContext(r.Context())
		})
		handler := server.RequestID(server.Timeout(time.Second)(server.Timeout(time.Minute)(inner)))

		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))

		assert.NotEmpty(t, id)
	})
}

// deadlineOf runs a request through the middlewares and returns the deadline the final handler sees.
func deadlineOf(t *testing.T, handler func(http.Handler) http.Handler, more ...func(http.Handler) http.Handler) (time.Time, bool) {
	t.Helper()

	var deadline time.Time
	var ok bool
	var final http.Handler = http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		deadline, ok = r.Context().Deadline()
	})
	for _, m := range slices.Backward(more) {
		final = m(final)
	}
	handler(final).ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
	return deadline, ok
}
