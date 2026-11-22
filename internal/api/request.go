package api

import (
	"bytes"
	jsonv1 "encoding/json"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
)

// decodeJSON reads and validates a JSON request body into a value of type T.
// On a malformed body it writes a 400, on validation failure a 422, and returns false.
func (a *API) decodeJSON[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var dst T
	if err := readJSON(r, &dst); err != nil {
		a.badRequestResponse(w, r, err)
		return dst, false
	}
	errs, err := a.validate(r.Context(), dst)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return dst, false
	}
	if errs != nil {
		a.failedValidationResponse(w, r, errs)
		return dst, false
	}
	return dst, true
}

// errEmptyBody reports a request without a body where JSON was expected.
var errEmptyBody = errors.New("body must not be empty")

// readJSON decodes a JSON request body within the server.BodyLimit of the route into dst,
// rejecting unknown members and duplicate names, with durations in nanoseconds.
func readJSON(r *http.Request, dst any) error {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		if maxBytesError, ok := errors.AsType[*http.MaxBytesError](err); ok {
			return fmt.Errorf("body must not be larger than %d bytes: %w", maxBytesError.Limit, maxBytesError)
		}
		return err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return errEmptyBody
	}

	err = json.Unmarshal(data, dst, json.RejectUnknownMembers(true), jsonv1.FormatDurationAsNano(true))
	if err == nil {
		return nil
	}
	if syntacticError, ok := errors.AsType[*jsontext.SyntacticError](err); ok {
		return describeSyntacticError(syntacticError)
	}
	if semanticError, ok := errors.AsType[*json.SemanticError](err); ok {
		return describeSemanticError(semanticError)
	}
	return err
}

// describeSyntacticError turns a JSON syntax error into a message for the client.
func describeSyntacticError(err *jsontext.SyntacticError) error {
	switch {
	case strings.Contains(err.Err.Error(), "after top-level value"):
		return errors.New("body must only contain a single JSON value")
	case strings.Contains(err.Err.Error(), "duplicate object member name"):
		return fmt.Errorf("body contains duplicate key %q", pointerPath(err.JSONPointer))
	default:
		return fmt.Errorf("body contains badly-formed JSON (at character %d)", err.ByteOffset)
	}
}

// describeSemanticError turns a JSON value that does not fit the Go type into a message for the client.
func describeSemanticError(err *json.SemanticError) error {
	field := pointerPath(err.JSONPointer)
	switch {
	case errors.Is(err, json.ErrUnknownName):
		return fmt.Errorf("body contains unknown key %q", field)
	case field != "":
		return fmt.Errorf("body contains incorrect JSON type for field %q", field)
	default:
		return fmt.Errorf("body contains incorrect JSON type (at character %d)", err.ByteOffset)
	}
}

// pointerPath names the member a JSON pointer refers to as a dotted path like the field paths of validation errors.
func pointerPath(pointer jsontext.Pointer) string {
	return strings.Join(slices.Collect(pointer.Tokens()), ".")
}
