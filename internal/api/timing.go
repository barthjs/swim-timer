package api

import (
	"net/http"
)

// timingStatusGetHandler reports whether the timing device is connected and through which port.
//
//	@Summary		Get timing status
//	@Description	Returns the connection state of the timing device, the event the simulator plays, and the name of the recording it replays.
//	@Tags			timing
//	@Produce		json
//	@Success		200	{object}	TimingStatusResponse
//	@Router			/timing/status [get]
func (a *API) timingStatusGetHandler(w http.ResponseWriter, r *http.Request) {
	status := a.timingService.GetStatus()

	a.writeJSON(w, r, http.StatusOK, TimingStatusResponse{
		Connected:      status.Connected,
		Port:           status.Port,
		ConfiguredPort: status.ConfiguredPort,
		SimulateEvent:  status.SimulateEvent,
		ReplayFile:     status.ReplayFile,
	}, nil)
}

// TimingStatusResponse describes the connection to the timing device.
type TimingStatusResponse struct {
	Connected      bool   `json:"connected"`
	Port           string `json:"port"`
	ConfiguredPort string `json:"configuredPort"`
	SimulateEvent  int    `json:"simulateEvent"`
	ReplayFile     string `json:"replayFile"`
}
