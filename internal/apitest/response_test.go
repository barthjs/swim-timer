package apitest_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/barthjs/swim-timer/internal/apitest"
)

// bodyJSON is the body the passing assertions check.
const bodyJSON = `{"name":"a","count":2,"items":[{"id":1},{"id":2}],"meta":{"x":true},"errors":{"server.port":["min"]}}`

// failJSON is the body the failing assertions check.
const failJSON = `{"n":1,"list":[1]}`

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

// TestResponse_AssertStatus covers AssertStatus through the shortcuts for the single status codes.
func TestResponse_AssertStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		assert func(*apitest.Response) *apitest.Response
	}{
		{
			name:   "ok",
			status: http.StatusOK,
			assert: (*apitest.Response).AssertOK,
		},
		{
			name:   "created",
			status: http.StatusCreated,
			assert: (*apitest.Response).AssertCreated,
		},
		{
			name:   "no content",
			status: http.StatusNoContent,
			assert: (*apitest.Response).AssertNoContent,
		},
		{
			name:   "bad request",
			status: http.StatusBadRequest,
			assert: (*apitest.Response).AssertBadRequest,
		},
		{
			name:   "unauthorized",
			status: http.StatusUnauthorized,
			assert: (*apitest.Response).AssertUnauthorized,
		},
		{
			name:   "forbidden",
			status: http.StatusForbidden,
			assert: (*apitest.Response).AssertForbidden,
		},
		{
			name:   "not found",
			status: http.StatusNotFound,
			assert: (*apitest.Response).AssertNotFound,
		},
		{
			name:   "unprocessable",
			status: http.StatusUnprocessableEntity,
			assert: (*apitest.Response).AssertUnprocessable,
		},
		{
			name:   "too many requests",
			status: http.StatusTooManyRequests,
			assert: (*apitest.Response).AssertTooManyRequests,
		},
	}

	for _, tt := range tests {
		t.Run("passes on the "+tt.name+" status", func(t *testing.T) {
			t.Parallel()

			response := apitest.NewClient(t, respond(tt.status, "")).GET("/")

			tt.assert(response)
		})
	}

	for _, tt := range tests {
		t.Run("fails on a status other than "+tt.name, func(t *testing.T) {
			t.Parallel()

			mock := &recordingT{
				TB: t,
			}
			response := apitest.NewClient(mock, respond(http.StatusTeapot, "")).GET("/")

			tt.assert(response)

			assert.True(t, mock.failed, "a wrong status must fail")
		})
	}
}

func TestResponse_AssertHeader(t *testing.T) {
	t.Parallel()

	t.Run("passes on the header value", func(t *testing.T) {
		t.Parallel()

		response := apitest.NewClient(t, respond(http.StatusOK, bodyJSON)).GET("/")

		response.AssertHeader("X-Test", "yes")
	})

	t.Run("fails on a different header value", func(t *testing.T) {
		t.Parallel()

		mock := &recordingT{
			TB: t,
		}
		response := apitest.NewClient(mock, respond(http.StatusOK, failJSON)).GET("/")

		response.AssertHeader("X-Test", "no")

		assert.True(t, mock.failed)
	})
}

func TestResponse_AssertJSON(t *testing.T) {
	t.Parallel()

	t.Run("passes on an equal body", func(t *testing.T) {
		t.Parallel()

		response := apitest.NewClient(t, respond(http.StatusOK, bodyJSON)).GET("/")

		response.AssertJSON(map[string]any{
			"meta":   map[string]bool{"x": true},
			"items":  []map[string]int{{"id": 1}, {"id": 2}},
			"count":  2,
			"name":   "a",
			"errors": map[string][]string{"server.port": {"min"}},
		})
	})

	t.Run("fails on a different body", func(t *testing.T) {
		t.Parallel()

		mock := &recordingT{
			TB: t,
		}
		response := apitest.NewClient(mock, respond(http.StatusOK, failJSON)).GET("/")

		response.AssertJSON(map[string]int{"n": 2})

		assert.True(t, mock.failed)
	})
}

func TestResponse_AssertJSONPath(t *testing.T) {
	t.Parallel()

	t.Run("passes on the values at the paths", func(t *testing.T) {
		t.Parallel()

		response := apitest.NewClient(t, respond(http.StatusOK, bodyJSON)).GET("/")

		response.
			AssertJSONPath("name", "a").
			AssertJSONPath("count", 2).
			AssertJSONPath("items.1.id", 2).
			AssertJSONPath("meta", map[string]bool{"x": true}).
			AssertJSONPath(`errors.server\.port.0`, "min")
	})

	tests := []struct {
		name  string
		check func(*apitest.Response)
	}{
		{
			name:  "fails on a different value",
			check: func(r *apitest.Response) { r.AssertJSONPath("n", 2) },
		},
		{
			name:  "fails on a missing path",
			check: func(r *apitest.Response) { r.AssertJSONPath("x", 1) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock := &recordingT{
				TB: t,
			}
			response := apitest.NewClient(mock, respond(http.StatusOK, failJSON)).GET("/")

			tt.check(response)

			assert.True(t, mock.failed)
		})
	}
}

func TestResponse_AssertJSONPathMissing(t *testing.T) {
	t.Parallel()

	t.Run("passes on missing paths", func(t *testing.T) {
		t.Parallel()

		response := apitest.NewClient(t, respond(http.StatusOK, bodyJSON)).GET("/")

		response.
			AssertJSONPathMissing("errors.server.port").
			AssertJSONPathMissing("missing").
			AssertJSONPathMissing("items.5").
			AssertJSONPathMissing("items.x").
			AssertJSONPathMissing("name.deeper")
	})

	t.Run("passes on a body that is no json", func(t *testing.T) {
		t.Parallel()

		mock := &recordingT{
			TB: t,
		}
		response := apitest.NewClient(mock, respond(http.StatusOK, "not json")).GET("/")

		response.AssertJSONPathMissing("anything")

		assert.False(t, mock.failed)
	})

	t.Run("fails on a present path", func(t *testing.T) {
		t.Parallel()

		mock := &recordingT{
			TB: t,
		}
		response := apitest.NewClient(mock, respond(http.StatusOK, failJSON)).GET("/")

		response.AssertJSONPathMissing("n")

		assert.True(t, mock.failed)
	})
}

func TestResponse_AssertJSONCount(t *testing.T) {
	t.Parallel()

	t.Run("passes on the number of elements", func(t *testing.T) {
		t.Parallel()

		response := apitest.NewClient(t, respond(http.StatusOK, bodyJSON)).GET("/")

		response.AssertJSONCount("items", 2).AssertJSONCount("meta", 1).AssertJSONCount("", 5)
	})

	tests := []struct {
		name  string
		check func(*apitest.Response)
	}{
		{
			name:  "fails on a different number",
			check: func(r *apitest.Response) { r.AssertJSONCount("list", 3) },
		},
		{
			name:  "fails on a missing path",
			check: func(r *apitest.Response) { r.AssertJSONCount("x", 0) },
		},
		{
			name:  "fails on a scalar",
			check: func(r *apitest.Response) { r.AssertJSONCount("n", 1) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock := &recordingT{
				TB: t,
			}
			response := apitest.NewClient(mock, respond(http.StatusOK, failJSON)).GET("/")

			tt.check(response)

			assert.True(t, mock.failed)
		})
	}
}

func TestResponse_JSON(t *testing.T) {
	t.Parallel()

	mock := &recordingT{
		TB: t,
	}
	response := apitest.NewClient(mock, respond(http.StatusOK, "not json")).GET("/")

	decode := func() { response.JSON(&struct{}{}) }

	assert.Panics(t, decode)
	assert.True(t, mock.failed)
}

func TestResponse_Decode(t *testing.T) {
	t.Parallel()

	type payload struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	response := apitest.NewClient(t, respond(http.StatusOK, bodyJSON)).GET("/")

	decoded := response.Decode[payload]()

	assert.Equal(t, payload{
		Name:  "a",
		Count: 2,
	}, decoded)
}

// respond returns a handler that answers with the given status and body.
func respond(status int, body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Test", "yes")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	})
}
