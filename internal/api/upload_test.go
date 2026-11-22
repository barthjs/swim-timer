package api

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
)

// testUploadMaxSizeMB is the file size limit the upload tests pass to readUpload.
const testUploadMaxSizeMB = 1

// testUploadAllowedTypes are the media types the upload tests pass to readUpload, which the text of their files has.
var testUploadAllowedTypes = []string{"text/plain"}

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
