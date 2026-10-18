package api

import (
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/barthjs/swim-timer/internal/domain"
)

const (
	// maxRecordingUploadMB bounds the size of an uploaded recording, many days of a device sending without pause.
	maxRecordingUploadMB = 100
)

var (
	// allowedRecordingUploadTypes are the detected content types a recording may have.
	allowedRecordingUploadTypes = []string{"text/plain"}
)

// recordingListHandler lists every stored recording of the timing data.
//
//	@Summary		List recordings
//	@Description	Lists every stored recording of the timing data and marks the one being written (Admin only).
//	@Tags			recordings
//	@Produce		json
//	@Success		200	{array}		RecordingResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/recordings [get]
func (a *API) recordingListHandler(w http.ResponseWriter, r *http.Request) {
	recordings, err := a.recordingService.List(r.Context())
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	response := make([]RecordingResponse, 0, len(recordings))
	for _, recording := range recordings {
		response = append(response, toRecordingResponse(recording))
	}

	a.writeJSON(w, r, http.StatusOK, response, nil)
}

// RecordingResponse is a stored recording of the timing data.
type RecordingResponse struct {
	Name string `json:"name"`
	Path string `json:"path"`
	// Current reports whether the recording is being written right now.
	Current bool `json:"current"`
}

// recordingCreateHandler stores an uploaded recording.
//
//	@Summary		Create recording
//	@Description	Stores an uploaded recording of the timing data under its sanitized filename. A file that is not a recording is rejected (422), as is the name of the recording being written (409) (Admin only).
//	@Tags			recordings
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			file	formData	file	true	"Recording"
//	@Success		201		{object}	RecordingResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		409		{object}	ErrorResponse
//	@Failure		422		{object}	ValidationErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/recordings [post]
func (a *API) recordingCreateHandler(w http.ResponseWriter, r *http.Request) {
	data, filename, ok := a.readUpload(w, r, maxRecordingUploadMB, allowedRecordingUploadTypes)
	if !ok {
		return
	}

	recording, err := a.recordingService.Create(r.Context(), filename, data)
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	a.writeJSON(w, r, http.StatusCreated, toRecordingResponse(recording), nil)
}

// toRecordingResponse maps a recording to its response.
func toRecordingResponse(recording domain.Recording) RecordingResponse {
	return RecordingResponse{
		Name:    recording.Name,
		Path:    recording.Path,
		Current: recording.Current,
	}
}

// recordingDeleteHandler removes a stored recording by name.
//
//	@Summary		Delete recording
//	@Description	Removes a stored recording by name. The replayed recording and the one being written cannot be deleted (409) (Admin only).
//	@Tags			recordings
//	@Param			name	path	string	true	"Recording filename"
//	@Success		204
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		409	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/recordings/{name} [delete]
func (a *API) recordingDeleteHandler(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	if err := a.recordingService.Delete(r.Context(), name); err != nil {
		a.respondError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// recordingDownloadHandler downloads a stored recording as it is stored.
//
//	@Summary		Download recording
//	@Description	Downloads a stored recording of the timing data as JSON Lines (Admin only).
//	@Tags			recordings
//	@Produce		application/jsonl
//	@Param			name	path	string	true	"Recording filename"
//	@Success		200
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/recordings/{name} [get]
func (a *API) recordingDownloadHandler(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	reader, err := a.recordingService.Download(r.Context(), name)
	if err != nil {
		a.respondError(w, r, err)
		return
	}
	defer func() { _ = reader.Close() }()

	writeDownloadHeaders(w, "application/jsonl", name)
	if _, err := io.Copy(w, reader); err != nil {
		// The headers and part of the file may already be sent, so the client may only see a broken download
		a.logger.Warn().Err(err).Str("recording", name).Msg("failed to send the recording")
	}
}

// recordingExportHandler downloads a stored recording as a CSV table with the decoded frames.
//
//	@Summary		Export recording
//	@Description	Downloads a stored recording as a CSV table with one row per frame, discarded block, gap, session, and meet line (Admin only).
//	@Tags			recordings
//	@Produce		text/csv
//	@Param			name	path	string	true	"Recording filename"
//	@Success		200
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/recordings/{name}/csv [get]
func (a *API) recordingExportHandler(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	// The recording is opened before the headers, so an unknown recording still gets a proper 404
	reader, err := a.recordingService.Download(r.Context(), name)
	if err != nil {
		a.respondError(w, r, err)
		return
	}
	defer func() { _ = reader.Close() }()

	download := &downloadWriter{
		writer:      w,
		contentType: "text/csv",
		filename:    strings.TrimSuffix(name, ".jsonl") + ".csv",
	}
	if err := a.recordingService.ExportCSV(reader, download); err != nil {
		if !download.started {
			a.serverErrorResponse(w, r, err)
			return
		}
		// The headers and part of the table are already sent, so the client may only see a broken download
		a.logger.Warn().Err(err).Str("recording", name).Msg("failed to export the recording")
	}
}

// downloadWriter sets the download headers with the first write, so a failure before it still gets an error response.
type downloadWriter struct {
	writer      http.ResponseWriter
	contentType string
	filename    string
	started     bool
}

// Write sets the download headers before the first bytes and writes p.
func (d *downloadWriter) Write(p []byte) (int, error) {
	if !d.started {
		writeDownloadHeaders(d.writer, d.contentType, d.filename)
		d.started = true
	}
	return d.writer.Write(p)
}
