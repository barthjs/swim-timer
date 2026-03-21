package handlers

import (
	"sync"
	"time"
)

type attempt struct {
	count     int
	firstHit  time.Time
	lastHit   time.Time
	expiresAt time.Time
}

// RateLimiter is a simple rate limiter that tracks the number of hits for a given key.
type RateLimiter struct {
	mu       sync.Mutex
	attempts map[string]*attempt
}

// newRateLimiter creates a new RateLimiter with a cleanup goroutine.
func newRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		attempts: make(map[string]*attempt),
	}
	go rl.cleanupRoutine()
	return rl
}

// cleanupRoutine runs a goroutine that periodically clears expired keys from the rate limiter.
func (r *RateLimiter) cleanupRoutine() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		r.ClearExpired()
	}
}

// Hit increments the number of hits for the given key.
func (r *RateLimiter) Hit(key string, window time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	att, exists := r.attempts[key]

	if !exists {
		r.attempts[key] = &attempt{
			count:     1,
			firstHit:  now,
			lastHit:   now,
			expiresAt: now.Add(window),
		}
		return
	}

	if now.After(att.expiresAt) {
		r.attempts[key] = &attempt{
			count:     1,
			firstHit:  now,
			lastHit:   now,
			expiresAt: now.Add(window),
		}
		return
	}

	att.count++
	att.lastHit = now
}

// TooManyAttempts returns true if the key has been hit too many times.
func (r *RateLimiter) TooManyAttempts(key string, max int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	att, exists := r.attempts[key]
	if !exists {
		return false
	}

	if time.Now().After(att.expiresAt) {
		return false
	}

	return att.count > max
}

// AvailableIn returns the number of seconds until the key is accessible again.
func (r *RateLimiter) AvailableIn(key string) int64 {
	r.mu.Lock()
	defer r.mu.Unlock()

	att, exists := r.attempts[key]
	if !exists {
		return 0
	}

	if time.Now().After(att.expiresAt) {
		return 0
	}

	retryAfter := time.Until(att.expiresAt)
	if retryAfter < 0 {
		return 0
	}

	return int64(retryAfter.Seconds())
}

// Clear removes the key from the rate limiter.
func (r *RateLimiter) Clear(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.attempts, key)
}

// ClearExpired removes all expired keys from the rate limiter.
func (r *RateLimiter) ClearExpired() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	for key, att := range r.attempts {
		if now.After(att.expiresAt) {
			delete(r.attempts, key)
		}
	}
}
