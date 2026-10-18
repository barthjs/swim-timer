package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"strconv"

	"github.com/rs/zerolog"

	"github.com/barthjs/swim-timer/internal/apperr"
	"github.com/barthjs/swim-timer/internal/server"
	"github.com/barthjs/swim-timer/internal/service"
)

const (
	codeBadRequest             = "bad_request"
	codeInvalidCredentials     = "invalid_credentials"
	codeAuthenticationRequired = "authentication_required"
	codeForbidden              = "forbidden"
	codeNotFound               = "not_found"
	codeMethodNotAllowed       = "method_not_allowed"
	codeValidationFailed       = "validation_failed"
	codeConflict               = "conflict"
	codeRateLimited            = "rate_limited"
	codePayloadTooLarge        = "payload_too_large"
	codeTimeout                = "timeout"
	codeServerError            = "server_error"
)

// ErrorResponse is the JSON body for all non-validation error responses.
// RequestID matches the X-Request-ID header and the log lines of the request.
type ErrorResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId,omitempty"`
}

// ValidationErrorResponse is the JSON body for a 422 with per-field details.
type ValidationErrorResponse struct {
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	Errors    FieldErrors `json:"errors"`
	RequestID string      `json:"requestId,omitempty"`
}

// respondError is the single place that maps a shared domain/service error to its canonical HTTP response.
func (a *API) respondError(w http.ResponseWriter, r *http.Request, err error) {
	if validationErr, ok := errors.AsType[*apperr.ValidationError](err); ok {
		a.failedValidationResponse(w, r, validationErr.Fields)
		return
	}
	if invalidErr, ok := errors.AsType[*service.InvalidConfigError](err); ok {
		a.badRequestResponse(w, r, invalidErr)
		return
	}
	// A meet file that cannot be read names itself and the line, so the admin can fix the export
	if dataErr, ok := errors.AsType[*service.MeetDataError](err); ok {
		a.badRequestResponse(w, r, dataErr)
		return
	}

	switch {
	case errors.Is(err, apperr.ErrNotFound):
		a.notFoundResponse(w, r)
	case errors.Is(err, apperr.ErrConflict):
		a.conflictResponse(w, r, "the request conflicts with existing data")
	case errors.Is(err, apperr.ErrVersionConflict):
		a.conflictResponse(w, r, "the resource was modified concurrently, please retry")
	case errors.Is(err, service.ErrRecordingInUse):
		a.conflictResponse(w, r, "the recording is replayed or being written")
	default:
		a.serverErrorResponse(w, r, err)
	}
}

// errorResponse sends a JSON payload with the given status code, machine-readable code and message.
func (a *API) errorResponse(w http.ResponseWriter, r *http.Request, status int, code string, message string) {
	if err := writeJSON(w, status, ErrorResponse{
		Code:      code,
		Message:   message,
		RequestID: server.RequestIDFromContext(r.Context()),
	}, nil); err != nil {
		logRequestFields(a.logger.Error().Err(err), r).Str("trace", string(debug.Stack())).Msg("failed to write error response")

		w.WriteHeader(http.StatusInternalServerError)
	}
}

// badRequestResponse writes a 400 Bad Request response, or a 413 if the body exceeded the route's size limit.
func (a *API) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	if maxBytesErr, ok := errors.AsType[*http.MaxBytesError](err); ok {
		a.payloadTooLargeResponse(w, r, maxBytesErr.Limit)
		return
	}
	a.errorResponse(w, r, http.StatusBadRequest, codeBadRequest, err.Error())
}

// invalidCredentialsResponse writes a 401 Unauthorized response for failed authentication attempts.
func (a *API) invalidCredentialsResponse(w http.ResponseWriter, r *http.Request) {
	a.errorResponse(w, r, http.StatusUnauthorized, codeInvalidCredentials, "invalid authentication credentials")
}

// authenticationRequiredResponse writes a 401 Unauthorized response when no valid session is present.
func (a *API) authenticationRequiredResponse(w http.ResponseWriter, r *http.Request) {
	a.errorResponse(w, r, http.StatusUnauthorized, codeAuthenticationRequired, "authentication required")
}

// forbiddenResponse writes a 403 Forbidden response.
func (a *API) forbiddenResponse(w http.ResponseWriter, r *http.Request) {
	a.errorResponse(w, r, http.StatusForbidden, codeForbidden, "insufficient permissions")
}

// notFoundResponse writes a 404 Not Found response.
func (a *API) notFoundResponse(w http.ResponseWriter, r *http.Request) {
	a.errorResponse(w, r, http.StatusNotFound, codeNotFound, "the requested resource could not be found")
}

// methodNotAllowedResponse writes a 405 Method Not Allowed response.
func (a *API) methodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
	a.errorResponse(w, r, http.StatusMethodNotAllowed, codeMethodNotAllowed, "method not allowed")
}

// conflictResponse writes a 409 Conflict response.
func (a *API) conflictResponse(w http.ResponseWriter, r *http.Request, message string) {
	a.errorResponse(w, r, http.StatusConflict, codeConflict, message)
}

// payloadTooLargeResponse writes a 413 Content Too Large response.
func (a *API) payloadTooLargeResponse(w http.ResponseWriter, r *http.Request, limit int64) {
	a.errorResponse(w, r, http.StatusRequestEntityTooLarge, codePayloadTooLarge, fmt.Sprintf("body must not be larger than %d bytes", limit))
}

// failedValidationResponse writes a 422 Unprocessable Content response with detailed validation errors.
func (a *API) failedValidationResponse(w http.ResponseWriter, r *http.Request, fieldErrors FieldErrors) {
	if err := writeJSON(w, http.StatusUnprocessableEntity, ValidationErrorResponse{
		Code:      codeValidationFailed,
		Message:   "validation failed",
		Errors:    fieldErrors,
		RequestID: server.RequestIDFromContext(r.Context()),
	}, nil); err != nil {
		a.serverErrorResponse(w, r, err)
	}
}

// rateLimitExceededResponse writes a 429 Too Many Requests response with a retry header.
func (a *API) rateLimitExceededResponse(w http.ResponseWriter, r *http.Request, retryAfter int64) {
	w.Header().Set("Retry-After", strconv.FormatInt(retryAfter, 10))
	a.errorResponse(w, r, http.StatusTooManyRequests, codeRateLimited, fmt.Sprintf("rate limit exceeded, retry after %d seconds", retryAfter))
}

// serverErrorResponse logs an internal server error and returns a generic 500 JSON response.
func (a *API) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, context.DeadlineExceeded) {
		logRequestFields(a.logger.Warn().Err(err), r).Msg("request timed out")
		a.timeoutResponse(w, r)
		return
	}

	logRequestFields(a.logger.Error().Err(err), r).Str("trace", string(debug.Stack())).Msg("internal server error")

	a.errorResponse(w, r, http.StatusInternalServerError, codeServerError, "the server encountered a problem and could not process your request")
}

// timeoutResponse writes a 503 Service Unavailable response when a request ran out of time.
func (a *API) timeoutResponse(w http.ResponseWriter, r *http.Request) {
	a.errorResponse(w, r, http.StatusServiceUnavailable, codeTimeout, "the request took too long, please retry")
}

// logRequestFields adds the standard per-request fields to a zerolog event.
// The caller starts the event at the desired level and terminates it with Msg.
func logRequestFields(e *zerolog.Event, r *http.Request) *zerolog.Event {
	return e.
		Str("proto", r.Proto).
		Str("method", r.Method).
		Str("uri", r.URL.RequestURI()).
		Str("ip", r.RemoteAddr).
		Str("user-agent", r.UserAgent()).
		Str("request_id", server.RequestIDFromContext(r.Context()))
}
