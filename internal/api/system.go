package api

import (
	"net/http"
)

// metaGetHandler returns the public configuration of the application.
//
//	@Summary		Get metadata
//	@Description	Returns the public configuration of the application.
//	@Tags			system
//	@Produce		json
//	@Success		200	{object}	MetaResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/meta [get]
func (a *API) metaGetHandler(w http.ResponseWriter, r *http.Request) {
	a.writeJSON(w, r, http.StatusOK, MetaResponse{
		Name:     a.options.Name,
		Version:  a.health.Version(),
		Language: a.options.Language,
		Theme:    a.options.Theme,
	}, nil)
}

// MetaResponse exposes public, non-sensitive configuration.
type MetaResponse struct {
	Name     string        `json:"name"`
	Version  string        `json:"version"`
	Language string        `json:"language"`
	Theme    ThemeSettings `json:"theme"`
}

// pingGetHandler responds with "pong".
//
//	@Summary		Get ping
//	@Description	Returns "pong" if the API is up and running.
//	@Tags			system
//	@Produce		plain
//	@Success		200	{string}	string	"pong"
//	@Router			/ping [get]
func (a *API) pingGetHandler(w http.ResponseWriter, _ *http.Request) {
	writeText(w, http.StatusOK, "pong", nil)
}
