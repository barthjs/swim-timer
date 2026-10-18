package api

import (
	"net/http"
)

// realtimeConnectHandler opens the WebSocket connection of a client, which receives every message of the application.
//
//	@Summary		Connect realtime
//	@Description	Upgrades the request to a WebSocket that receives JSON messages with a type and optional data. A client receives the live heat on connect and every change of it afterward.
//	@Tags			realtime
//	@Success		101
//	@Failure		400	{string}	string	"Not a WebSocket upgrade"
//	@Router			/ws [get]
func (a *API) realtimeConnectHandler(w http.ResponseWriter, r *http.Request) {
	a.hub.Serve(w, r, nil)
}
