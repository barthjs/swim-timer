package api

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
)

// testUploadMaxSizeMB is the file size limit the upload tests pass to readUpload.
const testUploadMaxSizeMB = 1

// testUploadAllowedTypes are the media types the upload tests pass to readUpload, which the text of their files has.
var testUploadAllowedTypes = []string{"text/plain"}

// jsonTestRequest is a request body with the kinds of members the JSON helpers handle.
type jsonTestRequest struct {
	Name    string        `json:"name" validate:"required"`
	Inner   jsonTestInner `json:"inner"`
	Timeout time.Duration `json:"timeout"`
}

// jsonTestInner is a nested member of jsonTestRequest.
type jsonTestInner struct {
	Count int `json:"count"`
}

func TestAPI_DecodeJSON(t *testing.T) {
	t.Parallel()

	t.Run("returns the valid body", func(t *testing.T) {
		t.Parallel()
		name := apitest.Faker(t).Word()
		recorder := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(`{"name":"`+name+`"}`))

		got, ok := newBareAPI().decodeJSON[jsonTestRequest](recorder, request)

		require.True(t, ok, "body: %s", recorder.Body.String())
		assert.Equal(t, name, got.Name)
	})

	t.Run("rejects a malformed body", func(t *testing.T) {
		t.Parallel()
		recorder := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(`{"name":`))

		_, ok := newBareAPI().decodeJSON[jsonTestRequest](recorder, request)

		require.False(t, ok)
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
		assert.Equal(t, "bad_request", errorCode(t, recorder))
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()
		recorder := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(`{}`))

		_, ok := newBareAPI().decodeJSON[jsonTestRequest](recorder, request)

		require.False(t, ok)
		assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
		assert.Equal(t, "required", fieldErrorTag(t, recorder, "name"))
	})
}

func TestReadJSON(t *testing.T) {
	t.Parallel()

	t.Run("decodes the body", func(t *testing.T) {
		t.Parallel()
		f := apitest.Faker(t)
		want := jsonTestRequest{
			Name: f.Word(),
			Inner: jsonTestInner{
				Count: f.IntRange(1, 1000),
			},
			Timeout: time.Duration(f.IntRange(1, 1000)) * time.Second,
		}
		// Durations are nanoseconds, as encoding/json/v2 has no default for them
		body := fmt.Sprintf(`{"name":%q,"inner":{"count":%d},"timeout":%d}`, want.Name, want.Inner.Count, want.Timeout)
		request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(body))

		var got jsonTestRequest
		require.NoError(t, readJSON(request, &got))

		assert.Equal(t, want, got)
	})

	t.Run("describes a rejected body", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		word := f.Word()
		tests := []struct {
			name    string
			body    string
			message string
		}{
			{
				name:    "empty body",
				body:    "",
				message: "body must not be empty",
			},
			{
				name:    "whitespace only",
				body:    " \n",
				message: "body must not be empty",
			},
			{
				name:    "broken syntax",
				body:    `{"name":"` + word,
				message: "body contains badly-formed JSON",
			},
			{
				name:    "invalid character",
				body:    `{"name":}`,
				message: "body contains badly-formed JSON",
			},
			{
				name:    "two values",
				body:    `{"name":"` + word + `"}{}`,
				message: "body must only contain a single JSON value",
			},
			{
				name:    "duplicate key",
				body:    `{"name":"` + word + `","name":"` + word + `"}`,
				message: `body contains duplicate key "name"`,
			},
			{
				name:    "unknown key",
				body:    `{"name":"` + word + `","other":true}`,
				message: `body contains unknown key "other"`,
			},
			{
				name:    "key in other case",
				body:    `{"Name":"` + word + `"}`,
				message: `body contains unknown key "Name"`,
			},
			{
				name:    "wrong type",
				body:    `{"name":` + f.DigitN(4) + `}`,
				message: `body contains incorrect JSON type for field "name"`,
			},
			{
				name:    "nested field of wrong type",
				body:    `{"inner":{"count":"` + word + `"}}`,
				message: `body contains incorrect JSON type for field "inner.count"`,
			},
			{
				name:    "wrong top-level type",
				body:    `[]`,
				message: "body contains incorrect JSON type (at character",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(tt.body))

				var dst jsonTestRequest
				err := readJSON(request, &dst)

				assert.ErrorContains(t, err, tt.message)
			})
		}
	})

	t.Run("reports a body over the limit", func(t *testing.T) {
		t.Parallel()
		body := `{"name":"` + apitest.Faker(t).LetterN(64) + `"}`
		request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(body))
		request.Body = http.MaxBytesReader(httptest.NewRecorder(), request.Body, 16)

		var dst jsonTestRequest
		err := readJSON(request, &dst)

		_, tooLarge := errors.AsType[*http.MaxBytesError](err)
		assert.True(t, tooLarge, "the error keeps the cause, so the response is a 413: %v", err)
		assert.ErrorContains(t, err, "body must not be larger than 16 bytes")
	})
}

func TestAPI_ReadUpload(t *testing.T) {
	t.Parallel()

	t.Run("returns the file", func(t *testing.T) {
		t.Parallel()
		f := apitest.Faker(t)
		filename := f.Word() + ".txt"
		content := []byte(f.LoremIpsumSentence(8))
		recorder := httptest.NewRecorder()
		request := buildMultipartBody(t, "file", filename, content)

		data, gotFilename, ok := readTestUpload(newBareAPI(), recorder, request)

		require.True(t, ok, "body: %s", recorder.Body.String())
		assert.Equal(t, content, data)
		assert.Equal(t, filename, gotFilename)
	})

	t.Run("requires the file field", func(t *testing.T) {
		t.Parallel()
		f := apitest.Faker(t)
		recorder := httptest.NewRecorder()
		request := buildMultipartBody(t, "other", f.Word()+".txt", []byte(f.LoremIpsumSentence(8)))

		_, _, ok := readTestUpload(newBareAPI(), recorder, request)

		require.False(t, ok)
		assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
		assert.Equal(t, "required", fieldErrorTag(t, recorder, "file"))
	})

	t.Run("rejects an unsupported type", func(t *testing.T) {
		t.Parallel()
		f := apitest.Faker(t)
		// Bytes without a known signature are detected as application/octet-stream
		content := make([]byte, f.IntRange(1, 1000))
		recorder := httptest.NewRecorder()
		request := buildMultipartBody(t, "file", f.Word()+".txt", content)

		_, _, ok := readTestUpload(newBareAPI(), recorder, request)

		require.False(t, ok)
		assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
		assert.Equal(t, "unsupported_type", fieldErrorTag(t, recorder, "file"))
	})

	t.Run("rejects a file over the limit", func(t *testing.T) {
		t.Parallel()
		// The request stays within the headroom for the multipart framing, so only the file itself is too large
		content := bytes.Repeat([]byte("a"), testUploadMaxSizeMB<<20+1)
		recorder := httptest.NewRecorder()
		request := buildMultipartBody(t, "file", apitest.Faker(t).Word()+".txt", content)

		_, _, ok := readTestUpload(newBareAPI(), recorder, request)

		require.False(t, ok)
		assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
		assert.Equal(t, "too_large", fieldErrorTag(t, recorder, "file"))
	})

	t.Run("rejects a body over the limit", func(t *testing.T) {
		t.Parallel()
		content := bytes.Repeat([]byte("a"), testUploadMaxSizeMB<<20+multipartFramingHeadroom+1)
		recorder := httptest.NewRecorder()
		request := buildMultipartBody(t, "file", apitest.Faker(t).Word()+".txt", content)

		_, _, ok := readTestUpload(newBareAPI(), recorder, request)

		require.False(t, ok)
		assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
		assert.Equal(t, "too_large", fieldErrorTag(t, recorder, "file"))
	})

	t.Run("rejects a malformed form", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		boundary := f.LetterN(16)
		tests := []struct {
			name        string
			contentType string
			body        string
		}{
			{
				name:        "not multipart",
				contentType: "application/json",
				body:        `{"file":"` + f.Word() + `"}`,
			},
			{
				name:        "missing boundary",
				contentType: "multipart/form-data",
				body:        f.LoremIpsumSentence(4),
			},
			{
				name:        "unterminated part",
				contentType: "multipart/form-data; boundary=" + boundary,
				body:        "--" + boundary + "\r\nContent-Disposition: form-data; name=\"file\"\r\n\r\n" + f.Word(),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				recorder := httptest.NewRecorder()
				request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(tt.body))
				request.Header.Set("Content-Type", tt.contentType)

				_, _, ok := readTestUpload(newBareAPI(), recorder, request)

				require.False(t, ok)
				assert.Equal(t, http.StatusBadRequest, recorder.Code)
				assert.Equal(t, "bad_request", errorCode(t, recorder))
			})
		}
	})
}

func TestUploadTypeAllowed(t *testing.T) {
	t.Parallel()

	// Text is detected as "text/plain; charset=utf-8"
	data := []byte(apitest.Faker(t).LoremIpsumSentence(8))
	tests := []struct {
		name    string
		allowed []string
		want    bool
	}{
		{
			name:    "allows a media type without its parameters",
			allowed: []string{"text/plain"},
			want:    true,
		},
		{
			name:    "allows a media type in other case and with spaces",
			allowed: []string{"image/png", " Text/Plain "},
			want:    true,
		},
		{
			name:    "rejects another media type",
			allowed: []string{"image/png"},
			want:    false,
		},
		{
			name:    "rejects any type without allowed media types",
			allowed: nil,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, uploadTypeAllowed(data, tt.allowed))
		})
	}
}

func TestUploadBodyLimit(t *testing.T) {
	t.Parallel()

	t.Run("fits a file of the maximum size", func(t *testing.T) {
		t.Parallel()
		content := bytes.Repeat([]byte("a"), testUploadMaxSizeMB<<20)

		request := buildMultipartBody(t, "file", apitest.Faker(t).Word()+".txt", content)

		assert.LessOrEqual(t, request.ContentLength, uploadBodyLimit(testUploadMaxSizeMB))
	})
}

func TestQueryParser_Fail(t *testing.T) {
	t.Parallel()

	t.Run("keeps the first error", func(t *testing.T) {
		t.Parallel()
		f := apitest.Faker(t)
		values := url.Values{
			"first":  {f.LetterN(8)},
			"second": {f.LetterN(8)},
		}
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/?"+values.Encode(), nil)
		p := newQueryParser(request)

		p.optionalInt("first")
		p.optionalBool("second")

		assert.EqualError(t, p.err, "invalid query parameter first")
	})

	t.Run("skips the reads after an error", func(t *testing.T) {
		t.Parallel()
		f := apitest.Faker(t)
		values := url.Values{
			"invalid": {f.LetterN(8)},
			"valid":   {f.Word()},
		}
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/?"+values.Encode(), nil)
		p := newQueryParser(request)

		p.optionalInt("invalid")
		got := p.optionalString("valid")

		assert.Nil(t, got)
	})
}

func TestPathInt(t *testing.T) {
	t.Parallel()

	t.Run("returns the number", func(t *testing.T) {
		t.Parallel()
		number := apitest.Faker(t).IntRange(1, 1000)

		got, err := pathInt(pathParamRequest(t, "number", strconv.Itoa(number)), "number")

		require.NoError(t, err)
		assert.Equal(t, number, got)
	})

	t.Run("rejects an invalid value", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		tests := []struct {
			name string
			raw  string
			want string
		}{
			{
				name: "missing",
				raw:  "",
				want: "missing path parameter: number",
			},
			{
				name: "malformed",
				raw:  f.LetterN(8),
				want: "invalid path parameter: number",
			},
			{
				name: "zero",
				raw:  "0",
				want: "invalid path parameter: number",
			},
			{
				name: "negative",
				raw:  strconv.Itoa(-f.IntRange(1, 1000)),
				want: "invalid path parameter: number",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				_, err := pathInt(pathParamRequest(t, "number", tt.raw), "number")

				assert.EqualError(t, err, tt.want)
			})
		}
	})
}

func TestQueryOptionalStringParam(t *testing.T) {
	t.Parallel()

	t.Run("returns the trimmed value", func(t *testing.T) {
		t.Parallel()
		value := apitest.Faker(t).Word()

		got, err := queryOptionalStringParam(url.Values{"key": {" " + value + " "}}, "key")

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, value, *got)
	})

	t.Run("returns nil for a missing value", func(t *testing.T) {
		t.Parallel()

		got, err := queryOptionalStringParam(url.Values{"key": {" "}}, "key")

		require.NoError(t, err)
		assert.Nil(t, got)
	})
}

func TestQueryOptionalIntParam(t *testing.T) {
	t.Parallel()

	t.Run("returns the value", func(t *testing.T) {
		t.Parallel()
		value := apitest.Faker(t).IntRange(-1000, 1000)

		got, err := queryOptionalIntParam(url.Values{"key": {strconv.Itoa(value)}}, "key")

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, value, *got)
	})

	t.Run("returns nil for a missing value", func(t *testing.T) {
		t.Parallel()

		got, err := queryOptionalIntParam(url.Values{}, "key")

		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("rejects a malformed value", func(t *testing.T) {
		t.Parallel()

		_, err := queryOptionalIntParam(url.Values{"key": {apitest.Faker(t).LetterN(8)}}, "key")

		assert.EqualError(t, err, "invalid query parameter key")
	})
}

func TestQueryOptionalBoolParam(t *testing.T) {
	t.Parallel()

	t.Run("returns the value", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			raw  string
			want bool
		}{
			{
				raw:  "true",
				want: true,
			},
			{
				raw:  "1",
				want: true,
			},
			{
				raw:  "Yes",
				want: true,
			},
			{
				raw:  "false",
				want: false,
			},
			{
				raw:  "0",
				want: false,
			},
			{
				raw:  "No",
				want: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.raw, func(t *testing.T) {
				t.Parallel()

				got, err := queryOptionalBoolParam(url.Values{"key": {tt.raw}}, "key")

				require.NoError(t, err)
				require.NotNil(t, got)
				assert.Equal(t, tt.want, *got)
			})
		}
	})

	t.Run("returns nil for a missing value", func(t *testing.T) {
		t.Parallel()

		got, err := queryOptionalBoolParam(url.Values{}, "key")

		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("rejects a malformed value", func(t *testing.T) {
		t.Parallel()

		_, err := queryOptionalBoolParam(url.Values{"key": {apitest.Faker(t).LetterN(8)}}, "key")

		assert.EqualError(t, err, "invalid query parameter key")
	})
}

// newBareAPI builds an API without services, enough for the helpers that read requests and write responses.
func newBareAPI() *API {
	return &API{
		logger: zerolog.Nop(),
		now:    time.Now,
	}
}

// readTestUpload reads the upload of request with the limits of these tests.
func readTestUpload(a *API, recorder *httptest.ResponseRecorder, request *http.Request) ([]byte, string, bool) {
	return a.readUpload(recorder, request, testUploadMaxSizeMB, testUploadAllowedTypes)
}

// buildMultipartBody builds a multipart request with one file field.
func buildMultipartBody(t *testing.T, field, filename string, content []byte) *http.Request {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(field, filename)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

// errorCode extracts the error code from a recorded error response.
func errorCode(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()

	var response ErrorResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response), "body: %s", recorder.Body.String())

	return response.Code
}

// fieldErrorTag extracts the first validation error tag of a field from a recorded 422 response.
func fieldErrorTag(t *testing.T, recorder *httptest.ResponseRecorder, field string) string {
	t.Helper()

	var response ValidationErrorResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response), "body: %s", recorder.Body.String())
	require.NotEmpty(t, response.Errors[field], "no errors for field %q: %s", field, recorder.Body.String())

	return response.Errors[field][0].Tag
}

// pathParamRequest builds a request whose route holds the path parameter name with value.
func pathParamRequest(t *testing.T, name, value string) *http.Request {
	t.Helper()

	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add(name, value)
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	return request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
}
