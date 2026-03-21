package handlers

import (
	"net/http"
	"time"
)

func (h *Handlers) loginHandler(w http.ResponseWriter, r *http.Request) {
	key := "login:" + r.RemoteAddr
	h.rateLimiter.Hit(key, 1*time.Minute)

	if h.rateLimiter.TooManyAttempts(key, 5) {
		h.rateLimitExceededResponse(w, r, h.rateLimiter.AvailableIn(key))
		return
	}

	var input struct {
		Key string `json:"key"`
	}

	if err := readJSON(w, r, &input); err != nil {
		h.badRequestResponse(w, r, err)
		return
	}

	if h.cfg.Secret != input.Key {
		h.invalidCredentialsResponse(w, r)
		return
	}

	h.rateLimiter.Clear(key)

	if err := writeJSON(w, http.StatusOK, envelope{"message": "successfully logged in"}, nil); err != nil {
		h.serverErrorResponse(w, r, err)
	}
}
