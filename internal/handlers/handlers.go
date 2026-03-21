package handlers

import (
	"net/http"

	"github.com/rs/zerolog"

	"github.com/barthjs/swim-timer/internal/config"
	"github.com/barthjs/swim-timer/internal/store"
	"github.com/barthjs/swim-timer/internal/ws"
	"github.com/barthjs/swim-timer/web"
)

type envelope map[string]any

// Handlers contains all the dependencies for the HTTP handlers.
type Handlers struct {
	cfg         config.Config
	logger      zerolog.Logger
	raceStore   *store.RaceDataStore
	presetStore *store.PresetsStore
	hub         *ws.Hub
	rateLimiter *RateLimiter
}

// New creates a new Handlers instance.
func New(
	cfg config.Config,
	logger zerolog.Logger,
	raceStore *store.RaceDataStore,
	presetStore *store.PresetsStore,
	hub *ws.Hub,
) *Handlers {
	return &Handlers{
		cfg:         cfg,
		logger:      logger,
		raceStore:   raceStore,
		presetStore: presetStore,
		hub:         hub,
		rateLimiter: newRateLimiter(),
	}
}

// indexHandler serves the index.html for the web spa.
func (h *Handlers) indexHandler(w http.ResponseWriter, _ *http.Request) {
	rawFile, _ := web.Files.ReadFile("dist/index.html")
	_, _ = w.Write(rawFile)
}

// pingHandler responds with "pong".
func (h *Handlers) pingHandler(w http.ResponseWriter, _ *http.Request) {
	writeText(w, http.StatusOK, "pong", nil)
}

// webSocketHandler upgrades an incoming HTTP request to a WebSocket connection
// and registers it with the Hub.
func (h *Handlers) webSocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := h.hub.Upgrade(w, r)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to upgrade to websocket")
		return
	}
	h.hub.HandleConnection(conn)
}
