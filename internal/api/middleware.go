package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// recoverPanic catches panics in downstream handlers and converts them to HTTP 500 responses.
func (a *API) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				a.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// logRequest logs incoming HTTP requests.
func (a *API) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logRequestFields(a.logger.Info(), r).Msg("")

		next.ServeHTTP(w, r)
	})
}

// cacheHeaderMiddleware sets headers to cache assets.
func (a *API) cacheHeaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")

		next.ServeHTTP(w, r)
	})
}

// rateLimitPerIP limits requests per resolved client IP.
func (a *API) rateLimitPerIP(name string, limit int, window time.Duration) func(http.Handler) http.Handler {
	return a.rateLimit(limit, window, func(r *http.Request) string { return name + ":ip:" + r.RemoteAddr })
}

// rateLimit enforces a rate limit per computed key.
func (a *API) rateLimit(limit int, window time.Duration, keyFunc func(r *http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFunc(r)
			if !a.enforceRateLimit(w, r, key, limit, window) {
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// requireGuest rejects the request if the current session is already authenticated.
func (a *API) requireGuest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.authService.Authenticated(r.Context()) {
			a.conflictResponse(w, r, "already authenticated")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// requireAuthentication ensures the current session is an authenticated session.
func (a *API) requireAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.authService.Authenticated(r.Context()) {
			a.authenticationRequiredResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// enforceRateLimit atomically counts an attempt for the given rate key
// and rejects the request if that pushed it over the limit.
func (a *API) enforceRateLimit(w http.ResponseWriter, r *http.Request, rateKey string, max int, window time.Duration) bool {
	allowed, retryAfter, err := a.rateLimiter.Allow(r.Context(), rateKey, max, window)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return false
	}
	a.setRateLimitHeaders(w, r, rateKey, max)
	if !allowed {
		a.rateLimitExceededResponse(w, r, retryAfter)
		return false
	}
	return true
}

// setRateLimitHeaders reports the limit, the remaining attempts, and the seconds until the window resets
// in the RateLimit-* headers, so clients can slow down before they are rejected.
// A limiter error leaves the headers out, the limit itself was already enforced.
func (a *API) setRateLimitHeaders(w http.ResponseWriter, r *http.Request, rateKey string, max int) {
	remaining, err := a.rateLimiter.Remaining(r.Context(), rateKey, max)
	if err != nil {
		return
	}
	w.Header().Set("RateLimit-Limit", strconv.Itoa(max))
	w.Header().Set("RateLimit-Remaining", strconv.Itoa(remaining))
	w.Header().Set("RateLimit-Reset", strconv.FormatInt(a.rateLimiter.AvailableIn(r.Context(), rateKey), 10))
}
