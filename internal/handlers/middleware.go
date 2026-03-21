package handlers

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// recoverPanic recovers from panics in downstream handlers and returns a 500 response.
func (h *Handlers) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// "Connection: close" header acts as a trigger to make Go's HTTP server automatically
				// close the current connection after a response has been sent.
				w.Header().Set("Connection", "close")
				h.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// setRequestIP normalizes the request IP by extracting the host part from the RemoteAddr.
func (h *Handlers) setRequestIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			// If the request is coming from proxy, RemoteAddr may be only an ip address
			next.ServeHTTP(w, r)
			return
		}
		r.RemoteAddr = ip

		next.ServeHTTP(w, r)
	})
}

// commonHeaders sets standard security headers.
func (h *Handlers) commonHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Referrer-Policy", "origin-when-cross-origin")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "deny")

		next.ServeHTTP(w, r)
	})
}

// logRequest logs incoming HTTP requests.
func (h *Handlers) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.logger.Info().
			Str("proto", r.Proto).
			Str("method", r.Method).
			Str("uri", r.URL.RequestURI()).
			Str("ip", r.RemoteAddr).
			Str("user-agent", r.UserAgent()).
			Msg("")

		next.ServeHTTP(w, r)
	})
}

// rateLimit applies a simple rate limit based on a key.
func (h *Handlers) rateLimit(
	limit int,
	window time.Duration,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.RemoteAddr
			h.rateLimiter.Hit(key, window)

			if h.rateLimiter.TooManyAttempts(key, limit) {
				h.rateLimitExceededResponse(w, r, h.rateLimiter.AvailableIn(key))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// cacheHeaderMiddleware sets appropriate headers to cache assets.
func (h *Handlers) cacheHeaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")

		next.ServeHTTP(w, r)
	})
}

// requireAuth checks if the request has a valid Authorization Bearer token.
func (h *Handlers) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token != h.cfg.Secret {
			h.invalidCredentialsResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}
