package api

import (
	"net/http"
	"time"
)

const (
	loginFailedRateLimit       = 5
	loginFailedRateLimitWindow = 15 * time.Minute
)

// authStatusGetHandler returns the authentication status of the current session.
//
//	@Summary		Get auth status
//	@Description	Returns whether the current session is authenticated.
//	@Tags			auth
//	@Produce		json
//	@Success		200	{object}	AuthStatusResponse
//	@Router			/auth/me [get]
func (a *API) authStatusGetHandler(w http.ResponseWriter, r *http.Request) {
	a.writeJSON(w, r, http.StatusOK, AuthStatusResponse{
		Authenticated: a.authService.Authenticated(r.Context()),
	}, nil)
}

// AuthStatusResponse reports whether the session is logged in.
type AuthStatusResponse struct {
	Authenticated bool `json:"authenticated"`
}

// AuthLoginRequest holds the admin password of a login.
type AuthLoginRequest struct {
	Password string `json:"password"`
}

// authLoginHandler starts an admin session with the admin password.
//
//	@Summary		Log in
//	@Description	Checks the admin password and starts an admin session.
//	@Tags			auth
//	@Accept			json
//	@Param			request	body	AuthLoginRequest	true	"Login credentials"
//	@Success		204
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		409	{object}	ErrorResponse
//	@Failure		429	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/auth/login [post]
func (a *API) authLoginHandler(w http.ResponseWriter, r *http.Request) {
	rateKey := "auth:login:ip:" + r.RemoteAddr
	// The attempt counts before the password check, so parallel requests cannot pass the limit together
	if !a.enforceRateLimit(w, r, rateKey, loginFailedRateLimit, loginFailedRateLimitWindow) {
		return
	}

	req, ok := a.decodeJSON[AuthLoginRequest](w, r)
	if !ok {
		return
	}

	authenticated, err := a.authService.Login(r.Context(), req.Password)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}
	if !authenticated {
		a.invalidCredentialsResponse(w, r)
		return
	}

	_ = a.rateLimiter.Reset(r.Context(), rateKey)

	w.WriteHeader(http.StatusNoContent)
}

// authLogoutHandler ends the current session.
//
//	@Summary		Log out
//	@Description	Ends the current session.
//	@Tags			auth
//	@Success		204
//	@Failure		500	{object}	ErrorResponse
//	@Router			/auth/logout [post]
func (a *API) authLogoutHandler(w http.ResponseWriter, r *http.Request) {
	if err := a.authService.Logout(r.Context()); err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
