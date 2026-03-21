package handlers

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"strconv"
)

// errorResponse sends a JSON payload with the given status code and message.
func (h *Handlers) errorResponse(w http.ResponseWriter, r *http.Request, status int, message string) {
	if err := writeJSON(w, status, envelope{"message": message}, nil); err != nil {
		h.logger.Error().
			Err(err).
			Str("proto", r.Proto).
			Str("method", r.Method).
			Str("uri", r.URL.RequestURI()).
			Str("ip", r.RemoteAddr).
			Str("user-agent", r.UserAgent()).
			Str("trace", string(debug.Stack())).
			Msg("failed to write error response")

		w.WriteHeader(http.StatusInternalServerError)
	}
}

// serverErrorResponse logs an internal server error and returns a generic 500 JSON response.
func (h *Handlers) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	h.logger.Error().
		Err(err).
		Str("proto", r.Proto).
		Str("method", r.Method).
		Str("uri", r.URL.RequestURI()).
		Str("ip", r.RemoteAddr).
		Str("user-agent", r.UserAgent()).
		Str("trace", string(debug.Stack())).
		Msg("internal server error")

	h.errorResponse(w, r, http.StatusInternalServerError, "the server encountered a problem and could not process your request")
}

// badRequestResponse returns a 400 Bad Request response. Used when parsing the request body fails.
func (h *Handlers) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	h.errorResponse(w, r, http.StatusBadRequest, err.Error())
}

// invalidCredentialsResponse returns a 401 Unauthorized response when authentication fails.
func (h *Handlers) invalidCredentialsResponse(w http.ResponseWriter, r *http.Request) {
	h.errorResponse(w, r, http.StatusUnauthorized, "invalid authentication credentials")
}

// notFoundResponse returns a 404 Not Found response when the requested resource does not exist.
func (h *Handlers) notFoundResponse(w http.ResponseWriter, r *http.Request) {
	h.errorResponse(w, r, http.StatusNotFound, "the requested resource could not be found")
}

// methodNotAllowedResponse returns a 405 Method Not Allowed response when the HTTP method is unsupported.
func (h *Handlers) methodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
	h.errorResponse(w, r, http.StatusMethodNotAllowed, "method not allowed")
}

// failedValidationResponse returns a 422 Unprocessable Entity response when validation fails.
// The fieldErrors map contains field-specific validation errors.
func (h *Handlers) failedValidationResponse(w http.ResponseWriter, r *http.Request, fieldErrors Errors) {
	payload := envelope{
		"message": "validation error",
		"errors":  fieldErrors,
	}

	if err := writeJSON(w, http.StatusUnprocessableEntity, payload, nil); err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// rateLimitExceededResponse returns a 429 TooManyRequests response when rate limiting is exceeded.
func (h *Handlers) rateLimitExceededResponse(w http.ResponseWriter, r *http.Request, retryAfter int64) {
	w.Header().Set("Retry-After", strconv.FormatInt(retryAfter, 10))
	h.errorResponse(w, r, http.StatusTooManyRequests, fmt.Sprintf("rate limit exceeded, retry after %d seconds", retryAfter))
}
