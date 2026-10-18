package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/barthjs/swim-timer/internal/apitest"
	"github.com/barthjs/swim-timer/internal/apperr"
)

// TestAPI_RespondError covers the mappings that the feature tests do not reach, such as a concurrent modification
// that cannot be provoked reliably through a request.
func TestAPI_RespondError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{
			name:   "answers a concurrent modification with a conflict",
			err:    apperr.ErrVersionConflict,
			status: http.StatusConflict,
			code:   codeConflict,
		},
		{
			name:   "answers any other error with a server error",
			err:    errors.New("unknown"),
			status: http.StatusInternalServerError,
			code:   codeServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			recorder := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
			err := fmt.Errorf("%s: %w", apitest.Faker(t).Word(), tt.err)

			newBareAPI().respondError(recorder, request, err)

			assert.Equal(t, tt.status, recorder.Code)
			assert.Equal(t, tt.code, errorCode(t, recorder))
		})
	}
}

func TestAPI_ServerErrorResponse(t *testing.T) {
	t.Parallel()

	t.Run("answers with a server error", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		err := errors.New(apitest.Faker(t).Word())

		newBareAPI().serverErrorResponse(recorder, request, err)

		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
		assert.Equal(t, codeServerError, errorCode(t, recorder))
	})

	t.Run("answers a timed out request with a timeout", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		err := fmt.Errorf("%s: %w", apitest.Faker(t).Word(), context.DeadlineExceeded)

		newBareAPI().serverErrorResponse(recorder, request, err)

		assert.Equal(t, http.StatusServiceUnavailable, recorder.Code)
		assert.Equal(t, codeTimeout, errorCode(t, recorder))
	})
}
