package api

import (
	"encoding/json/v2"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

// newBareAPI builds an API without services, enough for the helpers that read requests and write responses.
func newBareAPI() *API {
	return &API{
		logger: zerolog.Nop(),
		now:    time.Now,
	}
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
