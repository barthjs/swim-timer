// Package session configures the HTTP session manager, which keeps the sessions in memory.
package session

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"
)

// DefaultLifetime is how long a session lasts at most when New gets no WithLifetime.
const DefaultLifetime = 168 * time.Hour

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

// WithIdleTimeout ends a session after this long without a request, before its lifetime is over.
// Every request extends it again. A non-positive duration disables the idle timeout.
func WithIdleTimeout(timeout time.Duration) Option {
	return func(sm *scs.SessionManager) {
		if timeout > 0 {
			sm.IdleTimeout = timeout
		}
	}
}

// WithRememberMe makes the session cookie end with the browser session unless the login asks
// to be remembered, see Remember. Without it, every session cookie lasts for the whole lifetime.
func WithRememberMe() Option {
	return func(sm *scs.SessionManager) {
		sm.Cookie.Persist = false
	}
}

// Remember decides at login whether the session outlives the browser session. With remember set,
// the cookie is kept until the session's lifetime or idle timeout ends. Without it, the browser drops the cookie
// when it is closed. It only has an effect on a manager built with WithRememberMe and must be called after
// the session token was renewed for the login.
func Remember(ctx context.Context, sm *scs.SessionManager, remember bool) {
	sm.RememberMe(ctx, remember)
}

// New creates a session manager that keeps the sessions in memory.
func New(secureCookie bool, opts ...Option) (*scs.SessionManager, error) {
	sm := scs.New()
	sm.Lifetime = DefaultLifetime
	sm.HashTokenInStore = true
	sm.Cookie.HttpOnly = true
	// Lax (not Strict) so the session cookie is sent on top-level navigations from other sites into the app,
	// such as a redirect back from a login provider. Lax still withholds the cookie from cross-site POST, PUT,
	// and DELETE requests, which keeps the session safe from CSRF.
	sm.Cookie.SameSite = http.SameSiteLaxMode
	sm.Cookie.Secure = secureCookie

	for _, opt := range opts {
		opt(sm)
	}

	sm.Store = newMemoryStore(memoryCleanupInterval)

	return sm, nil
}

// StopCleanup stops the goroutine with which the in-memory store removes expired sessions and waits until it returned.
// Call it when the manager is no longer used. Further calls do nothing.
func StopCleanup(sm *scs.SessionManager) {
	if store, ok := sm.Store.(*memoryStore); ok {
		store.stopCleanup()
	}
}

// memoryCleanupInterval is how often the in-memory store removes expired sessions.
const memoryCleanupInterval = time.Minute

// memoryStore keeps the sessions in memory and removes expired ones on a goroutine until StopCleanup stops it.
// The own cleanup of memstore cannot be stopped safely, since memstore creates its stop channel
// on the cleanup goroutine, so the store runs memstore without it and tracks the expiry of every session itself.
type memoryStore struct {
	*memstore.MemStore

	mu       sync.Mutex
	expiries map[string]time.Time

	stop     chan struct{}
	stopOnce sync.Once
	// done is closed once the cleanup goroutine returned.
	done chan struct{}
}

// newMemoryStore creates a memoryStore that removes expired sessions every interval.
func newMemoryStore(interval time.Duration) *memoryStore {
	s := &memoryStore{
		MemStore: memstore.NewWithCleanupInterval(0),
		expiries: make(map[string]time.Time),
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
	go s.runCleanup(interval)
	return s
}

// Commit stores a session and remembers its expiry for the cleanup.
func (s *memoryStore) Commit(token string, b []byte, expiry time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.MemStore.Commit(token, b, expiry); err != nil {
		return err
	}
	s.expiries[token] = expiry
	return nil
}

// Delete removes a session.
func (s *memoryStore) Delete(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.expiries, token)
	return s.MemStore.Delete(token)
}

// stopCleanup ends the cleanup goroutine and waits until it returned. Further calls do nothing.
func (s *memoryStore) stopCleanup() {
	s.stopOnce.Do(func() { close(s.stop) })
	<-s.done
}

// runCleanup removes expired sessions every interval until stopCleanup is called.
func (s *memoryStore) runCleanup(interval time.Duration) {
	defer close(s.done)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case now := <-ticker.C:
			s.deleteExpired(now)
		case <-s.stop:
			return
		}
	}
}

// deleteExpired removes the sessions whose expiry lies before now.
func (s *memoryStore) deleteExpired(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for token, expiry := range s.expiries {
		if expiry.Before(now) {
			_ = s.MemStore.Delete(token)
			delete(s.expiries, token)
		}
	}
}
