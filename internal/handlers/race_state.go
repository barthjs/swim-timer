package handlers

import (
	"net/http"

	"github.com/barthjs/swim-timer/internal/data"
)

// getRaceStateHandler returns the current race state
func (h *Handlers) getRaceStateHandler(w http.ResponseWriter, r *http.Request) {
	state := h.hub.GetState()
	if state == nil {
		state = &data.RaceState{
			Lanes: make(map[int]*data.Lane),
		}
	}

	if err := writeJSON(w, http.StatusOK, state, nil); err != nil {
		h.serverErrorResponse(w, r, err)
	}
}
