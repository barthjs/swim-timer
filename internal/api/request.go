package api

import (
	"bytes"
	jsonv1 "encoding/json"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

const (
	// multipartFramingHeadroom is added to the file size limit to allow for the multipart framing.
	multipartFramingHeadroom = 256 << 10
	// uploadTimeout gives uploads more time than the default request timeout, for slow connections.
	uploadTimeout = 5 * time.Minute
	// downloadTimeout gives the downloads of recordings, which can be large, more time as well.
	downloadTimeout = 10 * time.Minute
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

// readJSON decodes a JSON request body into dst. Unknown members and duplicate names are rejected,
// and member names must match their tags exactly. Durations are nanoseconds, as encoding/json/v2 has no default for them.
// The size of the body is bounded by the server.BodyLimit of the route,
// and exceeding it returns an error wrapping *http.MaxBytesError.
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

// pointerPath names the member a JSON pointer refers to as a dotted path, such as "storage.s3.bucket",
// which matches the field paths of validation errors.
func pointerPath(pointer jsontext.Pointer) string {
	return strings.Join(slices.Collect(pointer.Tokens()), ".")
}

// readUpload parses the multipart "file" field, enforces maxSizeMB, and verifies the detected content type
// is in allowedTypes. It returns the file bytes and the client-supplied filename. The request may carry only
// the one file, so its body is limited to that file, even on a route with a larger limit.
// On failure, it writes the response and returns false.
func (a *API) readUpload(w http.ResponseWriter, r *http.Request, maxSizeMB int, allowedTypes []string) ([]byte, string, bool) {
	maxFile := int64(maxSizeMB) << 20
	maxRequest := maxFile + multipartFramingHeadroom

	r.Body = http.MaxBytesReader(w, r.Body, maxRequest)
	if !a.parseMultipartForm(w, r, maxRequest, "file") {
		return nil, "", false
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		a.failedValidationResponse(w, r, FieldErrors{"file": {
			{
				Tag: "required",
			},
		}})
		return nil, "", false
	}
	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(io.LimitReader(file, maxFile+1))
	if err != nil {
		a.badRequestResponse(w, r, err)
		return nil, "", false
	}
	if int64(len(data)) > maxFile {
		a.failedValidationResponse(w, r, FieldErrors{"file": {
			{
				Tag: "too_large",
			},
		}})
		return nil, "", false
	}
	if !uploadTypeAllowed(data, allowedTypes) {
		a.failedValidationResponse(w, r, FieldErrors{"file": {
			{
				Tag: "unsupported_type",
			},
		}})
		return nil, "", false
	}

	return data, header.Filename, true
}

// parseMultipartForm parses the multipart form of a request whose body is limited to maxBytes.
// A body over the limit is a 422 with a too_large error on field, so the form shows it next to the file input,
// while a request that is not a well-formed multipart form is a 400.
// On failure, it writes the response and returns false.
func (a *API) parseMultipartForm(w http.ResponseWriter, r *http.Request, maxBytes int64, field string) bool {
	err := r.ParseMultipartForm(maxBytes)
	if err == nil {
		return true
	}

	_, bodyTooLarge := errors.AsType[*http.MaxBytesError](err)
	if bodyTooLarge || errors.Is(err, multipart.ErrMessageTooLarge) {
		a.failedValidationResponse(w, r, FieldErrors{field: {
			{
				Tag: "too_large",
			},
		}})
		return false
	}

	a.badRequestResponse(w, r, err)
	return false
}

// uploadTypeAllowed reports whether the detected content type of data is in allowedTypes.
func uploadTypeAllowed(data []byte, allowedTypes []string) bool {
	detected := strings.ToLower(strings.TrimSpace(http.DetectContentType(data)))
	// DetectContentType can append parameters, such as "text/plain; charset=utf-8", compare only the media type
	if i := strings.IndexByte(detected, ';'); i != -1 {
		detected = strings.TrimSpace(detected[:i])
	}

	for _, allowed := range allowedTypes {
		if detected == strings.ToLower(strings.TrimSpace(allowed)) {
			return true
		}
	}
	return false
}

// uploadBodyLimit is the body limit of a route that accepts files of up to maxSizeMB in total.
func uploadBodyLimit(maxSizeMB int) int64 {
	return int64(maxSizeMB)<<20 + multipartFramingHeadroom
}

// queryParser reads query parameters while remembering the first error.
// Once an error is recorded, later reads return zero values.
type queryParser struct {
	values url.Values
	err    error
}

// newQueryParser builds a queryParser over a request's query string.
func newQueryParser(r *http.Request) *queryParser {
	return &queryParser{
		values: r.URL.Query(),
	}
}

// optionalString reads an optional text query parameter.
func (p *queryParser) optionalString(key string) *string {
	if p.err != nil {
		return nil
	}
	v, err := queryOptionalStringParam(p.values, key)
	p.fail(err)
	return v
}

// optionalInt reads an optional integer query parameter.
func (p *queryParser) optionalInt(key string) *int {
	if p.err != nil {
		return nil
	}
	v, err := queryOptionalIntParam(p.values, key)
	p.fail(err)
	return v
}

// optionalBool reads an optional boolean query parameter.
func (p *queryParser) optionalBool(key string) *bool {
	if p.err != nil {
		return nil
	}
	v, err := queryOptionalBoolParam(p.values, key)
	p.fail(err)
	return v
}

// fail records err as the first error, unless one is already set.
func (p *queryParser) fail(err error) {
	if p.err == nil {
		p.err = err
	}
}

// pathInt parses a positive integer path parameter. A missing or malformed value yields an error the handler turns
// into a 400 response.
func pathInt(r *http.Request, name string) (int, error) {
	raw := chi.URLParam(r, name)
	if raw == "" {
		return 0, fmt.Errorf("missing path parameter: %s", name)
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("invalid path parameter: %s", name)
	}
	return value, nil
}

// queryOptionalStringParam parses an optional string query parameter.
func queryOptionalStringParam(values url.Values, key string) (*string, error) {
	raw := strings.TrimSpace(values.Get(key))
	if raw == "" {
		return nil, nil
	}

	return &raw, nil
}

// queryOptionalIntParam parses an optional integer query parameter.
func queryOptionalIntParam(values url.Values, key string) (*int, error) {
	raw := strings.TrimSpace(values.Get(key))
	if raw == "" {
		return nil, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid query parameter %s", key)
	}

	return &value, nil
}

// queryOptionalBoolParam parses an optional boolean query parameter.
func queryOptionalBoolParam(values url.Values, key string) (*bool, error) {
	raw := strings.TrimSpace(values.Get(key))
	if raw == "" {
		return nil, nil
	}

	switch strings.ToLower(raw) {
	case "true", "1", "yes":
		return new(true), nil
	case "false", "0", "no":
		return new(false), nil
	default:
		return nil, fmt.Errorf("invalid query parameter %s", key)
	}
}
