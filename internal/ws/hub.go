package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"

	"github.com/barthjs/swim-timer/internal/data"
)

const (
	// pongWait is the time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// pingPeriod defines how often ping messages are sent to the peer.
	// Must be less than pongWait to ensure the connection stays alive.
	pingPeriod = (pongWait * 9) / 10

	// writeWait is the time allowed to write a message to the peer.
	writeWait = 10 * time.Second
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// client represents a single WebSocket connection and its associated data.
type client struct {
	hub  *Hub
	conn *websocket.Conn
	// send is a channel through which outgoing messages are passed to the writePump.
	send chan []byte
}

// Hub coordinates all WebSocket clients, handling their registration, unregistration,
// and the broadcasting of updates to all connected clients.
type Hub struct {
	logger zerolog.Logger
	// clients map the connected clients to a boolean value.
	clients map[*client]bool
	// register is a channel for clients that wish to join the hub.
	register chan *client
	// unregister is a channel for clients that wish to leave the hub.
	unregister chan *client
	// broadcast is a channel for messages to be sent to all connected clients.
	broadcast chan []byte
	// mu protects the state from concurrent access.
	mu sync.RWMutex
	// state holds the current race data that newly connected clients will receive.
	state *data.RaceState
}

func NewHub(
	logger zerolog.Logger,
) *Hub {
	return &Hub{
		logger:     logger,
		clients:    make(map[*client]bool),
		register:   make(chan *client),
		unregister: make(chan *client),
		broadcast:  make(chan []byte, 256),
		state: &data.RaceState{
			Lanes: make(map[int]*data.Lane),
		},
	}
}

// Run is the main event loop for the Hub. It uses channels to safely manage
// multiple concurrent WebSocket connections and coordinate message broadcasting.
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			// Hub is shutting down.
			return
		case client := <-h.register:
			// A new client has connected. Add it to the clients map and send the current state.
			h.mu.Lock()
			h.clients[client] = true
			if h.state != nil {
				msg, err := json.Marshal(wsMessage{Type: wsMessageTypeInit, Data: h.state})
				if err != nil {
					h.logger.Error().Err(err).Msg("failed to marshal initial state")
					h.mu.Unlock()
					continue
				}
				client.send <- msg
			}
			h.mu.Unlock()
		case client := <-h.unregister:
			// A client has disconnected. Remove it from the hub and close its "send" channel.
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			// A new update message has been received. Broadcast it to all connected clients.
			h.mu.RLock()
			var slow []*client
			for client := range h.clients {
				select {
				case client.send <- message:
					// Message sent successfully to the client's writePump
				default:
					// The client's "send" channel is full, meaning the connection might be slow.
					// Disconnect the client to prevent blocking the entire Hub.
					slow = append(slow, client)
				}
			}
			h.mu.RUnlock()
			if len(slow) > 0 {
				h.mu.Lock()
				for _, client := range slow {
					close(client.send)
					delete(h.clients, client)
				}
				h.mu.Unlock()
			}
		}
	}
}

// UpdateState updates the internal race state and broadcasts it to all clients.
func (h *Hub) UpdateState(state *data.RaceState) {
	msg, err := json.Marshal(wsMessage{Type: wsMessageTypeUpdate, Data: state})
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to marshal update state")
		return
	}

	h.mu.Lock()
	h.state = state
	h.mu.Unlock()

	h.broadcast <- msg
}

// GetState returns a deep copy of the current race state.
func (h *Hub) GetState() *data.RaceState {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.state == nil {
		return nil
	}
	return h.state.Copy()
}

// Upgrade upgrades the HTTP connection to a WebSocket connection.
func (h *Hub) Upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	return upgrader.Upgrade(w, r, nil)
}

// HandleConnection registers a new WebSocket connection and starts the read/write pumps.
func (h *Hub) HandleConnection(conn *websocket.Conn) {
	client := &client{
		hub:  h,
		conn: conn,
		send: make(chan []byte, 256),
	}
	h.register <- client

	go client.writePump()
	go client.readPump()
}

// readPump pumps messages from the WebSocket connection to the hub.
// For this application, it mainly serves to detect when a client has disconnected.
func (c *client) readPump() {
	defer func() {
		c.hub.unregister <- c
		_ = c.conn.Close()
	}()
	c.conn.SetReadLimit(512)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		// ReadMessage blocks until a message is received or the connection is closed.
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// writePump pumps messages from the hub's "send" channel to the WebSocket connection.
// This ensures that all WebSocket writes happen in a single goroutine per client.
func (c *client) writePump() {
	ticker := time.NewTicker(pingPeriod)
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
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)
			if err := w.Close(); err != nil {
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
