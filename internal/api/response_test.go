package api

import (
	"mime"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
)

func TestAPI_WriteJSON(t *testing.T) {
	t.Parallel()

	t.Run("reports an encoding failure as a server error", func(t *testing.T) {
		t.Parallel()
		recorder := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)

		// A channel has no JSON representation
		newBareAPI().writeJSON(recorder, request, http.StatusOK, make(chan int), nil)

		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
		assert.Equal(t, "server_error", errorCode(t, recorder))
	})
}

func TestWriteJSON(t *testing.T) {
	t.Parallel()

	t.Run("writes the status and headers", func(t *testing.T) {
		t.Parallel()
		location := "/" + apitest.Faker(t).Word()
		recorder := httptest.NewRecorder()

		require.NoError(t, writeJSON(recorder, http.StatusCreated, struct{}{}, http.Header{
			"Location": {location},
		}))

		assert.Equal(t, http.StatusCreated, recorder.Code)
		assert.Equal(t, location, recorder.Header().Get("Location"))
		assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
	})

	tests := []struct {
		name string
		data any
		body string
	}{
		{
			name: "encodes a nil slice as an empty array",
			data: struct {
				Items []string `json:"items"`
			}{},
			body: `{"items":[]}`,
		},
		{
			name: "encodes a nil map as an empty object",
			data: struct {
				Labels map[string]string `json:"labels"`
			}{},
			body: `{"labels":{}}`,
		},
		{
			name: "encodes a duration in nanoseconds",
			data: struct {
				Timeout time.Duration `json:"timeout"`
			}{
				Timeout: time.Second,
			},
			body: `{"timeout":1000000000}`,
		},
		{
			name: "drops a zero number with omitzero",
			data: struct {
				Count int `json:"count,omitzero"`
			}{},
			body: `{}`,
		},
		{
			name: "does not escape html",
			data: map[string]string{"text": "<b>&</b>"},
			body: `{"text":"<b>&</b>"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			recorder := httptest.NewRecorder()

			require.NoError(t, writeJSON(recorder, http.StatusOK, tt.data, nil))

			assert.Equal(t, tt.body, recorder.Body.String())
			assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
		})
	}
}

func TestWriteText(t *testing.T) {
	t.Parallel()

	t.Run("writes the text with status and headers", func(t *testing.T) {
		t.Parallel()
		f := apitest.Faker(t)
		text := f.LoremIpsumSentence(8)
		location := "/" + f.Word()
		recorder := httptest.NewRecorder()

		writeText(recorder, http.StatusCreated, text, http.Header{
			"Location": {location},
		})

		assert.Equal(t, http.StatusCreated, recorder.Code)
		assert.Equal(t, text, recorder.Body.String())
		assert.Equal(t, location, recorder.Header().Get("Location"))
		assert.Equal(t, "text/plain", recorder.Header().Get("Content-Type"))
	})
}

func TestWriteDownloadHeaders(t *testing.T) {
	t.Parallel()

	t.Run("marks the response as a download that is not cached", func(t *testing.T) {
		t.Parallel()
		name := apitest.Faker(t).LetterN(8) + ".txt"
		recorder := httptest.NewRecorder()

		writeDownloadHeaders(recorder, "text/plain", name)

		assert.Equal(t, "text/plain", recorder.Header().Get("Content-Type"))
		assert.Equal(t, "attachment; filename="+name, recorder.Header().Get("Content-Disposition"))
		assert.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))
	})

	t.Run("encodes a file name with special characters", func(t *testing.T) {
		t.Parallel()
		recorder := httptest.NewRecorder()

		writeDownloadHeaders(recorder, "text/plain", `a"b ü.txt`)

		_, params, err := mime.ParseMediaType(recorder.Header().Get("Content-Disposition"))
		require.NoError(t, err)
		assert.Equal(t, `a"b ü.txt`, params["filename"])
	})
}
