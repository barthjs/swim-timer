package api

import (
	"net/http"
)

// realtimeConnectHandler opens the WebSocket connection of a client, which receives every message of the application.
//
//	@Summary		Connect realtime
//	@Description	Upgrades the request to a WebSocket that receives JSON messages with a type and optional data. On connect, a client receives hello with the version and a hash of the settings, health with the state of every dependency, and init with the live heat. Afterward it receives update for every change of the live heat, health when a dependency changes, and changed with the resource and the item when the meet data, a preset, or the status of the timing device change.
//	@Tags			realtime
//	@Success		101
//	@Failure		400	{string}	string	"Not a WebSocket upgrade"
//	@Router			/ws [get]
func (a *API) realtimeConnectHandler(w http.ResponseWriter, r *http.Request) {
	a.hub.Serve(w, r, nil)
}
