package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

const (
	maxJSONSize      = 1 << 20  // 1MB
	maxMultipartSize = 10 << 20 // 10MB
)

// readJSON decodes a JSON request body into dst.
func readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONSize)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(dst)
	if err != nil {
		if syntaxError, ok := errors.AsType[*json.SyntaxError](err); ok {
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)
		}
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return errors.New("body contains badly-formed JSON")
		}
		if unmarshalTypeError, ok := errors.AsType[*json.UnmarshalTypeError](err); ok {
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)
		}
		if errors.Is(err, io.EOF) {
			return errors.New("body must not be empty")
		}
		if fieldName, ok := strings.CutPrefix(err.Error(), "json: unknown field "); ok {
			return fmt.Errorf("body contains unknown key %s", fieldName)
		}
		if maxBytesError, ok := errors.AsType[*http.MaxBytesError](err); ok {
			return fmt.Errorf("body must not be larger than %d bytes", maxBytesError.Limit)
		}
		if invalidUnmarshalError, ok := errors.AsType[*json.InvalidUnmarshalError](err); ok {
			panic(invalidUnmarshalError)
		}
		return err
	}

	// Ensure only a single JSON value is present.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("body must only contain a single JSON value")
	}

	return nil
}

// readFormFile extracts a file from a multipart form.
func readFormFile(r *http.Request, fieldName string) (multipart.File, *multipart.FileHeader, error) {
	file, header, err := r.FormFile(fieldName)
	if err != nil {
		return nil, nil, err
	}
	return file, header, nil
}
