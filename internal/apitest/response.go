package apitest

import (
	jsonv1 "encoding/json"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Response is a recorded response with fluent assertions. Every assertion reports the request
// and the body on failure, so a failing feature test shows what the handler answered.
type Response struct {
	t      testing.TB
	method string
	path   string

	Status int
	Header http.Header
	Body   []byte
}

// newResponse creates a Response from what the handler wrote to the recorder.
func newResponse(t testing.TB, method, path string, recorder *httptest.ResponseRecorder) *Response {
	return &Response{
		t:      t,
		method: method,
		path:   path,
		Status: recorder.Code,
		Header: recorder.Header(),
		Body:   recorder.Body.Bytes(),
	}
}

// AssertStatus checks the status code.
func (r *Response) AssertStatus(code int) *Response {
	r.t.Helper()
	assert.Equal(r.t, code, r.Status, r.describe())
	return r
}

// AssertOK checks for 200 OK.
func (r *Response) AssertOK() *Response {
	r.t.Helper()
	return r.AssertStatus(http.StatusOK)
}

// AssertCreated checks for 201 Created.
func (r *Response) AssertCreated() *Response {
	r.t.Helper()
	return r.AssertStatus(http.StatusCreated)
}

// AssertNoContent checks for 204 No Content.
func (r *Response) AssertNoContent() *Response {
	r.t.Helper()
	return r.AssertStatus(http.StatusNoContent)
}

// AssertBadRequest checks for 400 Bad Request.
func (r *Response) AssertBadRequest() *Response {
	r.t.Helper()
	return r.AssertStatus(http.StatusBadRequest)
}

// AssertUnauthorized checks for 401 Unauthorized.
func (r *Response) AssertUnauthorized() *Response {
	r.t.Helper()
	return r.AssertStatus(http.StatusUnauthorized)
}

// AssertForbidden checks for 403 Forbidden.
func (r *Response) AssertForbidden() *Response {
	r.t.Helper()
	return r.AssertStatus(http.StatusForbidden)
}

// AssertNotFound checks for 404 Not Found.
func (r *Response) AssertNotFound() *Response {
	r.t.Helper()
	return r.AssertStatus(http.StatusNotFound)
}

// AssertUnprocessable checks for 422 Unprocessable Entity.
func (r *Response) AssertUnprocessable() *Response {
	r.t.Helper()
	return r.AssertStatus(http.StatusUnprocessableEntity)
}

// AssertTooManyRequests checks for 429 Too Many Requests.
func (r *Response) AssertTooManyRequests() *Response {
	r.t.Helper()
	return r.AssertStatus(http.StatusTooManyRequests)
}

// AssertHeader checks that a response header has the value.
func (r *Response) AssertHeader(key, value string) *Response {
	r.t.Helper()
	assert.Equal(r.t, value, r.Header.Get(key), r.describe())
	return r
}

// AssertJSON checks that the body equals expected once both are encoded as JSON,
// so field order, whitespace, and the Go type of expected do not matter.
func (r *Response) AssertJSON(expected any) *Response {
	r.t.Helper()

	data, err := json.Marshal(expected, jsonv1.FormatDurationAsNano(true))
	require.NoError(r.t, err, "encoding the expected body")
	assert.JSONEq(r.t, string(data), string(r.Body), r.describe())
	return r
}

// AssertJSONPath checks the value at a dot-separated path into the body, such as "errors.0.field".
// Numbers in the path index arrays. A key that contains a dot is written with the dot escaped,
// such as `errors.server\.port` for the key "server.port". The value is compared after a JSON round trip,
// so an int matches the float64 the decoder produces.
func (r *Response) AssertJSONPath(path string, expected any) *Response {
	r.t.Helper()

	actual, ok := r.lookup(path)
	if !assert.True(r.t, ok, "no value at %q in %s", path, r.describe()) {
		return r
	}
	assert.Equal(r.t, normalize(r.t, expected), actual, "value at %q in %s", path, r.describe())
	return r
}

// AssertJSONPathMissing checks that the body has no value at the path.
func (r *Response) AssertJSONPathMissing(path string) *Response {
	r.t.Helper()

	_, ok := r.lookup(path)
	assert.False(r.t, ok, "unexpected value at %q in %s", path, r.describe())
	return r
}

// AssertJSONCount checks the number of elements of the array or object at the path, "" for the root.
func (r *Response) AssertJSONCount(path string, count int) *Response {
	r.t.Helper()

	actual, ok := r.lookup(path)
	if !assert.True(r.t, ok, "no value at %q in %s", path, r.describe()) {
		return r
	}
	switch value := actual.(type) {
	case []any:
		assert.Len(r.t, value, count, r.describe())
	case map[string]any:
		assert.Len(r.t, value, count, r.describe())
	default:
		assert.Fail(r.t, "not an array or object", "value at %q in %s", path, r.describe())
	}
	return r
}

// JSON decodes the body into dst and fails the test if it is not valid JSON for dst.
func (r *Response) JSON(dst any) *Response {
	r.t.Helper()
	require.NoError(r.t, json.Unmarshal(r.Body, dst, jsonv1.FormatDurationAsNano(true)), r.describe())
	return r
}

// describe summarizes the request and the response for assertion messages.
func (r *Response) describe() string {
	return fmt.Sprintf("%s %s answered %d: %s", r.method, r.path, r.Status, r.Body)
}

// Decode decodes the body into a new value of type T, for example response.Decode[UserResponse]().
func (r *Response) Decode[T any]() T {
	r.t.Helper()

	var value T
	r.JSON(&value)
	return value
}

// lookup decodes the body and returns the value at a dot-separated path, see AssertJSONPath.
func (r *Response) lookup(path string) (any, bool) {
	r.t.Helper()

	var current any
	if err := json.Unmarshal(r.Body, &current); err != nil {
		return nil, false
	}
	if path == "" {
		return current, true
	}

	for _, segment := range splitPath(path) {
		switch value := current.(type) {
		case map[string]any:
			next, ok := value[segment]
			if !ok {
				return nil, false
			}
			current = next
		case []any:
			index, err := strconv.Atoi(segment)
			if err != nil || index < 0 || index >= len(value) {
				return nil, false
			}
			current = value[index]
		default:
			return nil, false
		}
	}
	return current, true
}

// splitPath splits a path at its dots, keeping escaped dots ("\\.") inside a segment.
func splitPath(path string) []string {
	const placeholder = "\x00"

	segments := strings.Split(strings.ReplaceAll(path, `\.`, placeholder), ".")
	for i, segment := range segments {
		segments[i] = strings.ReplaceAll(segment, placeholder, ".")
	}
	return segments
}

// normalize converts a value to what decoding its JSON into an any produces.
func normalize(t testing.TB, value any) any {
	t.Helper()

	data, err := json.Marshal(value, jsonv1.FormatDurationAsNano(true))
	require.NoError(t, err)

	var normalized any
	require.NoError(t, json.Unmarshal(data, &normalized))
	return normalized
}
