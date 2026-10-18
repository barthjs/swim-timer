package api

import (
	"net/http"

	"github.com/barthjs/swim-timer/internal/service"
)

// updateStatusGetHandler returns the state of the last update.
//
//	@Summary		Get update status
//	@Description	Returns the state of the last update started on this instance (Admin only).
//	@Tags			update
//	@Produce		json
//	@Success		200	{object}	UpdateStatusResponse
//	@Failure		401	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/update/status [get]
func (a *API) updateStatusGetHandler(w http.ResponseWriter, r *http.Request) {
	status := a.updateService.Status()
	a.writeJSON(w, r, http.StatusOK, UpdateStatusResponse{
		State:   string(status.State),
		Version: status.Version,
		Error:   status.Error,
	}, nil)
}

// UpdateStatusResponse is the state of the last update.
type UpdateStatusResponse struct {
	State   string `json:"state" enums:"idle,running,failed,applied"`
	Version string `json:"version,omitzero"`
	Error   string `json:"error,omitzero"`
}

// updateCheckHandler compares the running version with the latest release.
//
//	@Summary		Check update
//	@Description	Compares the running version with the latest release and reports whether this installation can install it itself (Admin only).
//	@Tags			update
//	@Produce		json
//	@Success		200	{object}	UpdateCheckResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/update [get]
func (a *API) updateCheckHandler(w http.ResponseWriter, r *http.Request) {
	check, err := a.updateService.Check(r.Context())
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	a.writeJSON(w, r, http.StatusOK, UpdateCheckResponse{
		Current:     check.Current,
		Latest:      check.Latest,
		URL:         check.URL,
		Newer:       check.Newer,
		Unsupported: check.Unsupported,
	}, nil)
}

// UpdateCheckResponse compares the running version with the latest release.
// Unsupported names why this installation cannot install it itself, empty if it can.
type UpdateCheckResponse struct {
	Current     string `json:"current"`
	Latest      string `json:"latest"`
	URL         string `json:"url"`
	Newer       bool   `json:"newer"`
	Unsupported string `json:"unsupported,omitzero" enums:"development_build,container,package_managed,not_writable"`
}

// updateStartHandler installs the latest release in the background. The application restarts once it is installed.
//
//	@Summary		Start update
//	@Description	Starts installing the latest release in the background and answers right away. The application restarts once it is installed, see the update status (Admin only).
//	@Tags			update
//	@Success		202
//	@Failure		401	{object}	ErrorResponse
//	@Failure		409	{object}	ErrorResponse
//	@Failure		422	{object}	ValidationErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/update [post]
func (a *API) updateStartHandler(w http.ResponseWriter, r *http.Request) {
	if err := a.updateService.Start(); err != nil {
		a.respondError(w, r, err)
		return
	}

	// The update outlives the request that started it, the application waits for it before exiting
	a.background(service.UpdateTimeout, a.updateService.Run)

	w.WriteHeader(http.StatusAccepted)
}
