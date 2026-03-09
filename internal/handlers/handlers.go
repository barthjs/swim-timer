package handlers

import (
	"github.com/rs/zerolog"

	"github.com/barthjs/swim-timer/internal/config"
)

type envelope map[string]any

// Handlers contains all the dependencies for the HTTP handlers.
type Handlers struct {
	cfg    config.Config
	logger zerolog.Logger
}

// New creates a new Handlers instance.
func New(
	cfg config.Config,
	logger zerolog.Logger,
) *Handlers {
	return &Handlers{
		cfg:    cfg,
		logger: logger,
	}
}
