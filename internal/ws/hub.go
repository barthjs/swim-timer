// Package ws sends messages of the application to the connected WebSocket clients, to every client
// or to the clients of a topic, with an optional snapshot message that brings a newly connected client up to date.
//
// The caller assigns the topics of a client when it connects, for example from the credentials of the request,
// so a client never chooses what it receives.
package ws

import (
	"context"
	"iter"
	"maps"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

const (
	// defaultPongWait is the time allowed to read the next pong message from the peer.
	defaultPongWait = 60 * time.Second
	// writeWait is the time allowed to write a message to the peer.
	writeWait = 10 * time.Second
	// maxMessageSize bounds a message read from the peer. Clients only listen,
	// so anything larger is treated as misbehavior and closes the connection.
	maxMessageSize = 512
	// sendBufferSize is the number of queued messages after which a client counts as too slow and is dropped.
	sendBufferSize = 256
	// outboundBufferSize is the number of messages Broadcast, Publish, and Disconnect queue before they block the caller.
	outboundBufferSize = 256
)

// Option configures a Hub.
type Option func(*Hub)

// WithCheckOrigin sets the check of the Origin header of a connecting client. Without it every origin is accepted,
// which only fits a hub whose messages are public. A hub that assigns topics from cookies must check the origin,
// since a browser sends its cookies along with a connection opened by any other site.
func WithCheckOrigin(check func(r *http.Request) bool) Option {
	return func(h *Hub) {
		h.upgrader.CheckOrigin = check
	}
}

// WithPongWait sets how long a client may stay silent before it counts as disconnected.
// Pings are sent at nine tenths of it, so a healthy client always answers in time.
// Values of zero or less keep the default of one minute.
func WithPongWait(d time.Duration) Option {
	return func(h *Hub) {
		if d > 0 {
			h.pongWait = d
		}
	}
}

// SnapshotFunc returns the message a newly connected client receives first, or nil if there is nothing to send yet.
type SnapshotFunc func() ([]byte, error)

// client represents a single WebSocket connection.
type client struct {
	hub    *Hub
	conn   *websocket.Conn
	topics []string
	// send passes outgoing messages to the writePump.
	send chan []byte
}

// outbound is one message on its way to the clients of a topic, or to every client for the empty topic.
// A disconnect closes the clients instead of sending them a message.
type outbound struct {
	topic      string
	message    []byte
	disconnect bool
}

// Hub coordinates all WebSocket clients, handling their registration, unregistration,
// and the delivery of messages to every client or to the clients of a topic.
type Hub struct {
	logger   zerolog.Logger
	snapshot SnapshotFunc
	pongWait time.Duration
	upgrader websocket.Upgrader

	register   chan *client
	unregister chan *client
	outbound   chan outbound
	done       chan struct{}

	// started guards Run, since the done channel can only be closed once.
	started atomic.Bool

	mu      sync.RWMutex
	clients map[*client]bool
	topics  map[string]map[*client]struct{}
}

// New creates a Hub. The snapshot may be nil.
func New(logger zerolog.Logger, snapshot SnapshotFunc, opts ...Option) *Hub {
	h := &Hub{
		logger:   logger,
		snapshot: snapshot,
		pongWait: defaultPongWait,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(*http.Request) bool {
				return true
			},
		},
		register:   make(chan *client),
		unregister: make(chan *client),
		outbound:   make(chan outbound, outboundBufferSize),
		done:       make(chan struct{}),
		clients:    make(map[*client]bool),
		topics:     make(map[string]map[*client]struct{}),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// Run is the main event loop of the Hub. It serializes registration, unregistration,
// and the delivery of messages in one goroutine until ctx is canceled, then disconnects every client,
// since an HTTP server shutdown does not close hijacked WebSocket connections.
// It may only be called once per Hub, a second call logs an error and returns.
func (h *Hub) Run(ctx context.Context) {
	if !h.started.CompareAndSwap(false, true) {
		h.logger.Error().Msg("websocket hub is already running")
		return
	}
	defer func() {
		h.closeAll()
		close(h.done)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case c := <-h.register:
			h.addClient(c)
		case c := <-h.unregister:
			h.removeClient(c)
		case out := <-h.outbound:
			if out.disconnect {
				h.disconnect(out.topic)
			} else {
				h.fanOut(out.topic, out.message)
			}
		}
	}
}

// Broadcast queues a message for every connected client. It blocks while the outbound queue is full
// and returns immediately once the hub has stopped.
func (h *Hub) Broadcast(message []byte) {
	h.enqueue(outbound{
		message: message,
	})
}

// Publish queues a message for the clients of a topic. It blocks while the outbound queue is full
// and returns immediately once the hub has stopped.
func (h *Hub) Publish(topic string, message []byte) {
	h.enqueue(outbound{
		topic:   topic,
		message: message,
	})
}

// Disconnect queues the closing of every client of a topic, for example when the credentials behind it end.
// It blocks while the outbound queue is full and returns immediately once the hub has stopped.
func (h *Hub) Disconnect(topic string) {
	h.enqueue(outbound{
		topic:      topic,
		disconnect: true,
	})
}

// TryPublish queues a message for the clients of a topic like Publish, but reports false instead of blocking
// while the outbound queue is full. A caller that must never wait, such as the reader of a message stream,
// uses it and decides itself what a dropped message means.
func (h *Hub) TryPublish(topic string, message []byte) bool {
	return h.tryEnqueue(outbound{
		topic:   topic,
		message: message,
	})
}

// TryDisconnect queues the closing of the clients of a topic like Disconnect, but reports false instead of blocking
// while the outbound queue is full.
func (h *Hub) TryDisconnect(topic string) bool {
	return h.tryEnqueue(outbound{
		topic:      topic,
		disconnect: true,
	})
}

// tryEnqueue hands an outbound message to Run without waiting. A stopped hub drops the message and reports success,
// since there is no client left to miss it.
func (h *Hub) tryEnqueue(out outbound) bool {
	select {
	case h.outbound <- out:
		return true
	case <-h.done:
		return true
	default:
		return false
	}
}

// enqueue hands an outbound message to Run, or drops it once the hub has stopped.
func (h *Hub) enqueue(out outbound) {
	select {
	case h.outbound <- out:
	case <-h.done:
	}
}

// ServeHTTP upgrades the request to a WebSocket connection and registers it as a client without topics.
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.Serve(w, r, nil)
}

// Serve upgrades the request to a WebSocket connection and registers it as a client of the topics.
func (h *Hub) Serve(w http.ResponseWriter, r *http.Request, topics []string) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		// The upgrader has already written an HTTP error response
		h.logger.Debug().Err(err).Msg("websocket upgrade failed")
		return
	}

	c := &client{
		hub:    h,
		conn:   conn,
		topics: topics,
		send:   make(chan []byte, sendBufferSize),
	}

	select {
	case h.register <- c:
	case <-h.done:
		_ = conn.Close()
		return
	}

	go c.readPump()
	go c.writePump()
}

// addClient adds a client to every client and to its topics and queues the snapshot for it.
func (h *Hub) addClient(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[c] = true
	for _, topic := range c.topics {
		if h.topics[topic] == nil {
			h.topics[topic] = make(map[*client]struct{})
		}
		h.topics[topic][c] = struct{}{}
	}

	if h.snapshot == nil {
		return
	}

	message, err := h.snapshot()
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to build websocket snapshot")
		return
	}
	if message != nil {
		c.send <- message
	}
}

// removeClient removes a client and closes its send channel, which stops its writePump.
func (h *Hub) removeClient(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.drop(c)
}

// closeAll removes every client. Closing a send channel makes its writePump send a close message
// and close the connection, which also ends its readPump.
func (h *Hub) closeAll() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for c := range h.clients {
		h.drop(c)
	}
}

// disconnect removes every client of a topic.
func (h *Hub) disconnect(topic string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for c := range h.topics[topic] {
		h.drop(c)
	}
}

// fanOut passes a message to every client, or to the clients of the topic if it is not empty. A client whose
// send channel is full is disconnected, so one slow connection cannot block all the others.
func (h *Hub) fanOut(topic string, message []byte) {
	h.mu.RLock()
	var slow []*client
	for c := range h.recipients(topic) {
		select {
		case c.send <- message:
		default:
			slow = append(slow, c)
		}
	}
	h.mu.RUnlock()

	if len(slow) == 0 {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	for _, c := range slow {
		h.drop(c)
	}
}

// recipients returns the clients a message for the topic reaches. The caller holds the lock.
func (h *Hub) recipients(topic string) iter.Seq[*client] {
	if topic == "" {
		return maps.Keys(h.clients)
	}
	return maps.Keys(h.topics[topic])
}

// drop removes a registered client from every client and from its topics and closes its send channel.
// A client that is no longer registered is left alone, since its send channel is already closed.
// The caller holds the lock.
func (h *Hub) drop(c *client) {
	if _, ok := h.clients[c]; !ok {
		return
	}

	delete(h.clients, c)
	for _, topic := range c.topics {
		delete(h.topics[topic], c)
		if len(h.topics[topic]) == 0 {
			delete(h.topics, topic)
		}
	}
	close(c.send)
}

// readPump reads from the connection only to process control frames and to detect when the client has disconnected.
func (c *client) readPump() {
	defer func() {
		select {
		case c.hub.unregister <- c:
		case <-c.hub.done:
		}
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(c.hub.pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(c.hub.pongWait))
		return nil
	})

	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

// writePump writes the queued messages and periodic pings to the connection.
// All writes happen in this goroutine, since a WebSocket connection supports only one concurrent writer.
func (c *client) writePump() {
	ticker := time.NewTicker(c.hub.pingPeriod())
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// pingPeriod is how often pings are sent, less than pongWait so the connection stays alive.
func (h *Hub) pingPeriod() time.Duration {
	return h.pongWait * 9 / 10
}
