package session

import (
	"sync"
	"time"

	"github.com/alexedwards/scs/v2/memstore"
)

// memoryCleanupInterval is how often the in-memory store removes expired sessions.
const memoryCleanupInterval = time.Minute

// memoryStore keeps the sessions in memory and removes expired ones on a goroutine until StopCleanup stops it.
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
		// The own cleanup of memstore cannot be stopped safely, since memstore creates its stop channel
		// on the cleanup goroutine, so the store tracks the expiry of every session itself.
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
