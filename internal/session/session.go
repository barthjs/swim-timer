// Package session configures the HTTP session manager, which keeps the sessions in memory.
package session

import (
	"context"
	"net/http"
	"time"

	"github.com/alexedwards/scs/v2"
)

// DefaultLifetime is how long a session lasts at most when New gets no WithLifetime.
const DefaultLifetime = 168 * time.Hour

// hostCookiePrefix is the prefix of a secure session cookie.
const hostCookiePrefix = "__Host-"

// Option configures the session manager built by New.
type Option func(*scs.SessionManager)

// WithLifetime sets the absolute lifetime of a session: it ends this long after it was created,
// however active the user is. A non-positive duration is ignored.
func WithLifetime(lifetime time.Duration) Option {
	return func(sm *scs.SessionManager) {
		if lifetime > 0 {
			sm.Lifetime = lifetime
		}
	}
}

// WithRememberMe makes the session cookie end with the browser session unless the login asks to be remembered,
// see Remember. Without it, every session cookie lasts for the whole lifetime.
func WithRememberMe() Option {
	return func(sm *scs.SessionManager) {
		sm.Cookie.Persist = false
	}
}

// Remember decides at login whether the cookie outlives the browser session, on a manager built with WithRememberMe
// and after the session token was renewed.
func Remember(ctx context.Context, sm *scs.SessionManager, remember bool) {
	sm.RememberMe(ctx, remember)
}

// New creates a session manager that keeps the sessions in memory.
// Its cookie carries the name, with the prefix __Host- when the cookie is secure.
func New(cookieName string, secureCookie bool, opts ...Option) (*scs.SessionManager, error) {
	sm := scs.New()
	sm.Lifetime = DefaultLifetime
	sm.HashTokenInStore = true
	sm.Cookie.HttpOnly = true
	// Lax (not Strict) so the session cookie is sent on top-level navigations from other sites into the app,
	// such as a redirect back from a login provider. Lax still withholds the cookie from cross-site POST, PUT, and DELETE requests,
	// which keeps the session safe from CSRF.
	sm.Cookie.SameSite = http.SameSiteLaxMode
	sm.Cookie.Secure = secureCookie
	sm.Cookie.Name = cookieName
	if secureCookie {
		sm.Cookie.Name = hostCookiePrefix + cookieName
	}

	for _, opt := range opts {
		opt(sm)
	}

	sm.Store = newMemoryStore(memoryCleanupInterval)

	return sm, nil
}

// StopCleanup stops the cleanup of the in-memory store and waits for it, once the manager is no longer used.
func StopCleanup(sm *scs.SessionManager) {
	if store, ok := sm.Store.(*memoryStore); ok {
		store.stopCleanup()
	}
}
