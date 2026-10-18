package ratelimiter

import (
	"context"
	"sync"
	"time"
)

// memoryEntry is a single key's counter and its expiry.
type memoryEntry struct {
	hits      int
	expiresAt time.Time
}

// memoryLimiter stores rate limit counters in process memory.
type memoryLimiter struct {
	mu      sync.Mutex
	entries map[string]memoryEntry
	now     func() time.Time
}

// newMemoryLimiter creates a memoryLimiter that reads the time from now.
func newMemoryLimiter(now func() time.Time) *memoryLimiter {
	return &memoryLimiter{
		entries: make(map[string]memoryEntry),
		now:     now,
	}
}

// Allow implements Limiter.
func (m *memoryLimiter) Allow(_ context.Context, key string, limit int, window time.Duration) (bool, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := m.now()
	entry, ok := m.entries[key]
	if ok && entry.alive(now) {
		entry.hits++
	} else {
		entry = memoryEntry{
			hits:      1,
			expiresAt: now.Add(window),
		}
	}
	m.entries[key] = entry

	if entry.hits > limit {
		return false, secondsUntil(now, entry.expiresAt), nil
	}

	return true, 0, nil
}

// TooManyAttempts implements Limiter.
func (m *memoryLimiter) TooManyAttempts(_ context.Context, key string, limit int) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.entries[key]
	if !ok || !entry.alive(m.now()) {
		return false, nil
	}

	return entry.hits >= limit, nil
}

// Hit implements Limiter.
func (m *memoryLimiter) Hit(_ context.Context, key string, window time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := m.now()
	entry, ok := m.entries[key]
	if ok && entry.alive(now) {
		entry.hits++
	} else {
		entry.hits = 1
	}
	entry.expiresAt = now.Add(window)
	m.entries[key] = entry

	return nil
}

// Remaining implements Limiter.
func (m *memoryLimiter) Remaining(_ context.Context, key string, limit int) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.entries[key]
	if !ok || !entry.alive(m.now()) {
		return limit, nil
	}

	return max(limit-entry.hits, 0), nil
}

// AvailableIn implements Limiter.
func (m *memoryLimiter) AvailableIn(_ context.Context, key string) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := m.now()
	entry, ok := m.entries[key]
	if !ok || !entry.alive(now) {
		return 0
	}

	return secondsUntil(now, entry.expiresAt)
}

// Reset implements Limiter.
func (m *memoryLimiter) Reset(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.entries, key)
	return nil
}

// ClearExpired implements Prunable.
func (m *memoryLimiter) ClearExpired(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := m.now()
	for key, entry := range m.entries {
		if !entry.alive(now) {
			delete(m.entries, key)
		}
	}
	return nil
}

// alive reports whether the entry's window is still open at now. A window closes at expiresAt itself.
func (e memoryEntry) alive(now time.Time) bool {
	return now.Before(e.expiresAt)
}

// secondsUntil returns the whole seconds from now until t, rounded up so a client never retries too early.
func secondsUntil(now, t time.Time) int64 {
	remaining := t.Sub(now)
	if remaining <= 0 {
		return 0
	}
	return int64((remaining + time.Second - 1) / time.Second)
}
