package service

import "sync"

// fakeBroadcaster keeps every broadcast message.
type fakeBroadcaster struct {
	mu       sync.Mutex
	messages [][]byte
}

// Broadcast records the message.
func (b *fakeBroadcaster) Broadcast(message []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.messages = append(b.messages, message)
}

// count returns the number of recorded messages.
func (b *fakeBroadcaster) count() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.messages)
}
