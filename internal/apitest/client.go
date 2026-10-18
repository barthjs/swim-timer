// Package apitest drives an HTTP handler in feature tests. It sends requests the way a browser would,
// keeps cookies between requests so sessions work, and offers fluent assertions on the response.
package apitest

import (
	"bytes"
	jsonv1 "encoding/json"
	"encoding/json/v2"
	"io"
	"maps"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

// baseURL is the origin requests are sent to. The handler is called directly, so it only scopes the cookie jar.
var baseURL = &url.URL{
	Scheme: "http",
	Host:   "example.test",
}

// Client sends requests to a handler. It is cheap to create, so every test builds its own,
// which keeps tests independent when they run in parallel.
type Client struct {
	t       testing.TB
	handler http.Handler
	jar     http.CookieJar
	headers http.Header
}

// NewClient creates a Client for handler with an empty cookie jar.
func NewClient(t testing.TB, handler http.Handler) *Client {
	t.Helper()

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)

	return &Client{
		t:       t,
		handler: handler,
		jar:     jar,
		headers: http.Header{"Accept": []string{"application/json"}},
	}
}

// WithHeader returns a client that sends the header with every request. The cookie jar is shared,
// so a session started by one client stays valid for the other.
func (c *Client) WithHeader(key, value string) *Client {
	headers := c.headers.Clone()
	headers.Set(key, value)

	return &Client{
		t:       c.t,
		handler: c.handler,
		jar:     c.jar,
		headers: headers,
	}
}

// GET sends a GET request.
func (c *Client) GET(path string) *Response {
	return c.Do(http.MethodGet, path, nil)
}

// POST sends a POST request with body, see Do.
func (c *Client) POST(path string, body any) *Response {
	return c.Do(http.MethodPost, path, body)
}

// PUT sends a PUT request with body, see Do.
func (c *Client) PUT(path string, body any) *Response {
	return c.Do(http.MethodPut, path, body)
}

// PATCH sends a PATCH request with body, see Do.
func (c *Client) PATCH(path string, body any) *Response {
	return c.Do(http.MethodPatch, path, body)
}

// DELETE sends a DELETE request.
func (c *Client) DELETE(path string) *Response {
	return c.Do(http.MethodDelete, path, nil)
}

// Do sends a request and records the response. A nil body sends none, a Multipart body is sent as multipart/form-data,
// an io.Reader is sent as it is, and any other value is encoded as JSON.
func (c *Client) Do(method, path string, body any) *Response {
	c.t.Helper()

	reader, contentType := c.encode(body)
	req := httptest.NewRequestWithContext(c.t.Context(), method, path, reader)
	maps.Copy(req.Header, c.headers.Clone())
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for _, cookie := range c.jar.Cookies(baseURL) {
		req.AddCookie(cookie)
	}

	recorder := httptest.NewRecorder()
	c.handler.ServeHTTP(recorder, req)

	result := recorder.Result()
	c.jar.SetCookies(baseURL, result.Cookies())

	return newResponse(c.t, method, path, recorder)
}

// encode turns a request body into a reader and the content type it is sent with.
func (c *Client) encode(body any) (io.Reader, string) {
	c.t.Helper()

	switch value := body.(type) {
	case nil:
		return nil, ""
	case Multipart:
		return value.encode(c.t)
	case io.Reader:
		return value, ""
	default:
		// A nil slice or map stands for null, so a test can send what a client leaves unset
		data, err := json.Marshal(value, jsonv1.FormatDurationAsNano(true), json.FormatNilSliceAsNull(true), json.FormatNilMapAsNull(true))
		require.NoError(c.t, err, "encoding the request body")
		return bytes.NewReader(data), "application/json"
	}
}

// File is a file field of a multipart request.
type File struct {
	Field   string
	Name    string
	Content []byte
}

// Multipart is a request body sent as multipart/form-data, as a browser sends an upload form.
type Multipart struct {
	Fields map[string]string
	Files  []File
}

// encode writes the fields and files as a multipart/form-data body.
func (m Multipart) encode(t testing.TB) (io.Reader, string) {
	t.Helper()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	for name, value := range m.Fields {
		require.NoError(t, writer.WriteField(name, value))
	}
	for _, file := range m.Files {
		part, err := writer.CreateFormFile(file.Field, file.Name)
		require.NoError(t, err)
		_, err = part.Write(file.Content)
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())

	return &buf, writer.FormDataContentType()
}
