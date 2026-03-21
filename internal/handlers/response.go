package handlers

import (
	"encoding/json"
	"net/http"
)

// writeJSON writes data as JSON with status code and optional headers.
func writeJSON(w http.ResponseWriter, status int, data any, headers http.Header) error {
	js, err := json.Marshal(data)
	if err != nil {
		return err
	}

	for key, value := range headers {
		w.Header()[key] = value
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(js)

	return nil
}

// writeText writes plain text with status code and optional headers.
func writeText(w http.ResponseWriter, status int, data string, headers http.Header) {
	for key, value := range headers {
		w.Header()[key] = value
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(data))
}
