package api

import (
	"net/http"

	"github.com/barthjs/swim-timer/internal/domain"
)

// liveHeatGetHandler returns the live heat, the same data the websocket publishes.
//
//	@Summary		Get live heat
//	@Description	Returns the heat the timing device currently reports. Overlays receive every change of it through the websocket at /api/ws.
//	@Tags			live
//	@Produce		json
//	@Success		200	{object}	domain.LiveHeat
//	@Router			/live [get]
func (a *API) liveHeatGetHandler(w http.ResponseWriter, r *http.Request) {
	// The explicitly typed variable is the file's only reference to the domain package.
	// Without it, the import would be unused, and swag could not resolve domain.LiveHeat in the annotation above.
	var heat *domain.LiveHeat = a.timingService.GetLiveHeat() //nolint:staticcheck // The explicit type keeps the import.

	a.writeJSON(w, r, http.StatusOK, heat, nil)
}
