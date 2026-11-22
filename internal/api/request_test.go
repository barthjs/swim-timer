package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
)

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
