package api

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

const (
	// multipartFramingHeadroom is added to the file size limit to allow for the multipart framing.
	multipartFramingHeadroom = 256 << 10
	// uploadTimeout gives uploads more time than the default request timeout, for slow connections.
	uploadTimeout = 5 * time.Minute
)

// readUpload returns the content and name of the only file of the multipart "file" field within maxSizeMB
// and of an allowed type, or writes the failure and returns false.
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

// parseMultipartForm parses the multipart form of a request within maxBytes, or writes the failure and returns false.
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
	// DetectContentType can append parameters, so only the media type is compared
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
