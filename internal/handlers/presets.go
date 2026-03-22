package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"

	"github.com/julienschmidt/httprouter"

	"github.com/barthjs/swim-timer/internal/data"
	"github.com/barthjs/swim-timer/internal/store"
)

// listPresetsHandler returns a list of all available presets.
func (h *Handlers) listPresetsHandler(w http.ResponseWriter, r *http.Request) {
	presets, err := h.presetStore.List()
	if err != nil {
		h.serverErrorResponse(w, r, err)
		return
	}

	if err := writeJSON(w, http.StatusOK, presets, nil); err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// getPresetHandler returns the contents of a specific preset.
func (h *Handlers) getPresetHandler(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	name := params.ByName("name")
	if name == "" {
		h.notFoundResponse(w, r)
		return
	}

	preset, err := h.presetStore.Get(name)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, store.ErrInvalidPreset) {
			h.notFoundResponse(w, r)
			return
		}
		h.serverErrorResponse(w, r, err)
		return
	}

	if err := writeJSON(w, http.StatusOK, json.RawMessage(preset), nil); err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// savePresetHandler saves a new preset with the given name.
func (h *Handlers) savePresetHandler(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	name := params.ByName("name")
	if name == "" {
		h.failedValidationResponse(w, r, Errors{"name": Error{Rule: "required"}})
		return
	}

	var input data.Preset
	if err := readJSON(w, r, &input); err != nil {
		h.badRequestResponse(w, r, err)
		return
	}

	if errs := validate(input); errs != nil {
		h.failedValidationResponse(w, r, errs)
		return
	}

	if err := h.presetStore.Save(name, input); err != nil {
		if errors.Is(err, store.ErrInvalidPreset) {
			h.failedValidationResponse(w, r, Errors{"name": Error{Rule: "invalid"}})
			return
		}
		h.serverErrorResponse(w, r, err)
		return
	}

	if err := writeJSON(w, http.StatusOK, input, nil); err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// deletePresetHandler deletes a specific preset.
func (h *Handlers) deletePresetHandler(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	name := params.ByName("name")
	if name == "" {
		h.notFoundResponse(w, r)
		return
	}

	if err := h.presetStore.Delete(name); err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, store.ErrInvalidPreset) {
			h.notFoundResponse(w, r)
			return
		}
		h.serverErrorResponse(w, r, err)
		return
	}

	if err := writeJSON(w, http.StatusOK, envelope{"message": "preset successfully deleted"}, nil); err != nil {
		h.serverErrorResponse(w, r, err)
	}
}
