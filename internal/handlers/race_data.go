package handlers

import (
	"fmt"
	"mime/multipart"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// listRaceDataHandler returns a list of all available race data files.
func (h *Handlers) listRaceDataHandler(w http.ResponseWriter, r *http.Request) {
	statuses := h.raceStore.GetFileStatuses()

	if err := writeJSON(w, http.StatusOK, statuses, nil); err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// updateRaceDataHandler updates a specific race data file.
func (h *Handlers) updateRaceDataHandler(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	fileID := params.ByName("id")

	if !h.raceStore.IsAllowed(fileID) {
		h.notFoundResponse(w, r)
		return
	}

	if err := r.ParseMultipartForm(maxMultipartSize); err != nil {
		h.badRequestResponse(w, r, err)
		return
	}

	file, _, err := readFormFile(r, "file")
	if err != nil {
		h.badRequestResponse(w, r, err)
		return
	}
	defer func(file multipart.File) {
		_ = file.Close()
	}(file)

	if err := h.raceStore.SaveFile(fileID, file); err != nil {
		h.serverErrorResponse(w, r, err)
		return
	}

	if err := writeJSON(w, http.StatusOK, envelope{"message": "race data updated successfully"}, nil); err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// uploadMultipleRaceDataHandler uploads multiple race data files.
func (h *Handlers) uploadMultipleRaceDataHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxMultipartSize); err != nil {
		h.badRequestResponse(w, r, err)
		return
	}

	type fileResult struct {
		Filename string `json:"filename"`
		Matched  bool   `json:"matched"`
		Error    string `json:"error,omitempty"`
	}

	var results []fileResult

	form := r.MultipartForm
	if form == nil {
		h.badRequestResponse(w, r, fmt.Errorf("no multipart form data"))
		return
	}

	allFiles := make([]*multipart.FileHeader, 0)
	for _, headers := range form.File {
		allFiles = append(allFiles, headers...)
	}

	if len(allFiles) == 0 {
		h.badRequestResponse(w, r, fmt.Errorf("no files uploaded"))
		return
	}

	for _, fileHeader := range allFiles {
		res := fileResult{Filename: fileHeader.Filename}

		fileID, ok := h.raceStore.MatchFileID(fileHeader.Filename)
		if !ok {
			res.Matched = false
			results = append(results, res)
			continue
		}

		res.Matched = true

		file, err := fileHeader.Open()
		if err != nil {
			res.Error = fmt.Sprintf("failed to open: %v", err)
			results = append(results, res)
			continue
		}

		if err := h.raceStore.SaveFile(fileID, file); err != nil {
			_ = file.Close()
			res.Error = fmt.Sprintf("failed to save: %v", err)
			results = append(results, res)
			continue
		}
		_ = file.Close()

		results = append(results, res)
	}

	if err := writeJSON(w, http.StatusOK, results, nil); err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// reloadRaceDataHandler reloads all race data files in the race store.
func (h *Handlers) reloadRaceDataHandler(w http.ResponseWriter, r *http.Request) {
	if err := h.raceStore.LoadAll(); err != nil {
		h.errorResponse(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to reload race data: %v", err))
		return
	}

	if err := writeJSON(w, http.StatusOK, envelope{"message": "race data reloaded successfully"}, nil); err != nil {
		h.serverErrorResponse(w, r, err)
	}
}
