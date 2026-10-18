package ws_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/ws"
)

const readTimeout = 2 * time.Second

func TestWithCheckOrigin(t *testing.T) {
	t.Parallel()

	t.Run("accepts a connection from an allowed origin", func(t *testing.T) {
		t.Parallel()

		hub, server := startHub(t, nil, ws.WithCheckOrigin(allowOrigin("https://allowed.example")))

		conn := dialFrom(t, server, "https://allowed.example")

		broadcastUntilReceived(t, hub, conn, "ready")
	})

	t.Run("rejects a connection from another origin", func(t *testing.T) {
		t.Parallel()

		_, server := startHub(t, nil, ws.WithCheckOrigin(allowOrigin("https://allowed.example")))
		header := http.Header{}
		header.Set("Origin", "https://other.example")

		_, response, err := websocket.DefaultDialer.Dial(wsURL(server, ""), header)

		require.Error(t, err)
		require.NotNil(t, response)
		_ = response.Body.Close()
		assert.Equal(t, http.StatusForbidden, response.StatusCode)
	})
}

func TestWithPongWait(t *testing.T) {
	t.Parallel()

	t.Run("ignores a non-positive wait", func(t *testing.T) {
		t.Parallel()

		// A non-positive wait keeps the default, so the connection is not dropped right away
		hub, server := startHub(t, nil, ws.WithPongWait(0), ws.WithPongWait(-time.Second))
		conn := dial(t, server)
		broadcastUntilReceived(t, hub, conn, "ready")
		time.Sleep(100 * time.Millisecond)

		hub.Broadcast([]byte("still connected"))

		assert.Equal(t, "still connected", readUntil(t, conn, "still connected"))
	})
}

func TestHub_Run(t *testing.T) {
	t.Parallel()

	t.Run("closes the clients when stopped", func(t *testing.T) {
		t.Parallel()

		hub := ws.New(zerolog.Nop(), nil)
		ctx, cancel := context.WithCancel(t.Context())
		go hub.Run(ctx)
		server := httptest.NewServer(hub)
		t.Cleanup(server.Close)
		conn := dial(t, server)
		broadcastUntilReceived(t, hub, conn, "ready")

		cancel()

		assertClosed(t, conn)
	})

	t.Run("returns at once when already running", func(t *testing.T) {
		t.Parallel()

		hub, server := startHub(t, nil)
		conn := dial(t, server)
		// A delivered broadcast proves that the first Run is serving
		broadcastUntilReceived(t, hub, conn, "running")

		// A second Run returns right away instead of competing for the channels
		returned := make(chan struct{})
		go func() {
			hub.Run(t.Context())
			close(returned)
		}()

		select {
		case <-returned:
		case <-time.After(readTimeout):
			require.FailNow(t, "second Run did not return")
		}
		hub.Broadcast([]byte("still running"))
		assert.Equal(t, "still running", readUntil(t, conn, "still running"))
	})
}

func TestHub_Broadcast(t *testing.T) {
	t.Parallel()

	t.Run("reaches every client", func(t *testing.T) {
		t.Parallel()

		hub, server := startHub(t, nil)
		first := dial(t, server)
		second := dial(t, server)
		broadcastUntilReceived(t, hub, first, "first ready")
		broadcastUntilReceived(t, hub, second, "second ready")

		hub.Broadcast([]byte("hello"))

		assert.Equal(t, "hello", readUntil(t, first, "hello"))
		assert.Equal(t, "hello", readUntil(t, second, "hello"))
	})

	t.Run("keeps serving the remaining client after another one left", func(t *testing.T) {
		t.Parallel()

		hub, server := startHub(t, nil)
		leaving := dial(t, server)
		staying := dial(t, server)
		broadcastUntilReceived(t, hub, leaving, "ready")
		broadcastUntilReceived(t, hub, staying, "ready")
		require.NoError(t, leaving.Close())

		hub.Broadcast([]byte("after"))

		assert.Equal(t, "after", readUntil(t, staying, "after"))
	})

	t.Run("does not wait for a slow client", func(t *testing.T) {
		t.Parallel()

		hub, server := startHub(t, nil)
		slow := dial(t, server)
		broadcastUntilReceived(t, hub, slow, "slow ready")

		// The slow client never reads, so its socket and its queue fill up until the hub drops it.
		// Broadcasting must not wait for it.
		payload := []byte(strings.Repeat("x", 32*1024))
		done := make(chan struct{})
		go func() {
			for range 1000 {
				hub.Broadcast(payload)
			}
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(10 * time.Second):
			require.FailNow(t, "a slow client blocked the broadcast")
		}
		// The hub keeps serving new clients
		fresh := dial(t, server)
		broadcastUntilReceived(t, hub, fresh, "after")
	})

	t.Run("returns at once on a stopped hub", func(t *testing.T) {
		t.Parallel()

		hub := stoppedHub(t)

		done := make(chan struct{})
		go func() {
			for range 1000 {
				hub.Broadcast([]byte("dropped"))
			}
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(readTimeout):
			require.FailNow(t, "Broadcast blocked on a stopped hub")
		}
	})
}

func TestHub_Publish(t *testing.T) {
	t.Parallel()

	t.Run("reaches the clients of the topic only", func(t *testing.T) {
		t.Parallel()

		hub, server := startTopicHub(t)
		subscribed := dialTopics(t, server, "news")
		other := dialTopics(t, server, "sports")
		broadcastUntilReceived(t, hub, subscribed, "ready")
		broadcastUntilReceived(t, hub, other, "ready")

		hub.Publish("news", []byte("headline"))
		hub.Broadcast([]byte("end"))

		assert.Equal(t, "headline", readUntil(t, subscribed, "headline"))
		assert.NotContains(t, readAll(t, other, "end"), "headline")
	})

	t.Run("returns at once on a stopped hub", func(t *testing.T) {
		t.Parallel()

		hub := stoppedHub(t)

		done := make(chan struct{})
		go func() {
			for range 1000 {
				hub.Publish("news", []byte("dropped"))
			}
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(readTimeout):
			require.FailNow(t, "Publish blocked on a stopped hub")
		}
	})
}

func TestHub_Disconnect(t *testing.T) {
	t.Parallel()

	t.Run("closes the clients of the topic", func(t *testing.T) {
		t.Parallel()

		hub, server := startTopicHub(t)
		leaving := dialTopics(t, server, "alpha")
		staying := dialTopics(t, server, "beta")
		broadcastUntilReceived(t, hub, leaving, "ready")
		broadcastUntilReceived(t, hub, staying, "ready")

		hub.Disconnect("alpha")
		hub.Broadcast([]byte("after"))

		assertClosed(t, leaving)
		assert.Equal(t, "after", readUntil(t, staying, "after"))
	})
}

func TestHub_TryPublish(t *testing.T) {
	t.Parallel()

	t.Run("reaches the clients of the topic", func(t *testing.T) {
		t.Parallel()

		hub, server := startTopicHub(t)
		conn := dialTopics(t, server, "news")
		broadcastUntilReceived(t, hub, conn, "ready")

		queued := hub.TryPublish("news", []byte("headline"))

		assert.True(t, queued)
		assert.Equal(t, "headline", readUntil(t, conn, "headline"))
	})

	t.Run("reports a full outbound queue instead of waiting", func(t *testing.T) {
		t.Parallel()

		// A hub that does not run never drains its outbound queue
		hub := ws.New(zerolog.Nop(), nil)
		for range 256 {
			require.True(t, hub.TryPublish("news", []byte("queued")))
		}

		queued := hub.TryPublish("news", []byte("dropped"))

		assert.False(t, queued)
	})

	t.Run("accepts a message on a stopped hub", func(t *testing.T) {
		t.Parallel()

		hub := stoppedHub(t)

		queued := hub.TryPublish("news", []byte("dropped"))

		assert.True(t, queued, "a stopped hub has no client that misses the message")
	})
}

func TestHub_TryDisconnect(t *testing.T) {
	t.Parallel()

	t.Run("closes the clients of the topic", func(t *testing.T) {
		t.Parallel()

		hub, server := startTopicHub(t)
		conn := dialTopics(t, server, "alpha")
		broadcastUntilReceived(t, hub, conn, "ready")

		queued := hub.TryDisconnect("alpha")

		assert.True(t, queued)
		assertClosed(t, conn)
	})

	t.Run("reports a full outbound queue instead of waiting", func(t *testing.T) {
		t.Parallel()

		hub := ws.New(zerolog.Nop(), nil)
		for range 256 {
			require.True(t, hub.TryDisconnect("alpha"))
		}

		queued := hub.TryDisconnect("alpha")

		assert.False(t, queued)
	})
}

func TestHub_ServeHTTP(t *testing.T) {
	t.Parallel()

	t.Run("sends the snapshot on connect", func(t *testing.T) {
		t.Parallel()

		_, server := startHub(t, func() ([]byte, error) { return []byte(`{"type":"init"}`), nil })

		conn := dial(t, server)

		assert.Equal(t, `{"type":"init"}`, read(t, conn))
	})

	snapshotTests := []struct {
		name     string
		snapshot ws.SnapshotFunc
	}{
		{
			name:     "skips the snapshot without a snapshot function",
			snapshot: nil,
		},
		{
			name:     "skips an empty snapshot",
			snapshot: func() ([]byte, error) { return nil, nil },
		},
		{
			name:     "skips a failing snapshot",
			snapshot: func() ([]byte, error) { return nil, errors.New("not ready") },
		},
	}

	for _, tt := range snapshotTests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hub, server := startHub(t, tt.snapshot)
			conn := dial(t, server)

			// The first message the client sees is the broadcast, not a snapshot
			stop := make(chan struct{})
			defer close(stop)
			go func() {
				for {
					hub.Broadcast([]byte("update"))
					select {
					case <-stop:
						return
					case <-time.After(20 * time.Millisecond):
					}
				}
			}()

			assert.Equal(t, "update", read(t, conn))
		})
	}

	t.Run("rejects a request without websocket upgrade", func(t *testing.T) {
		t.Parallel()

		_, server := startHub(t, nil)
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
		require.NoError(t, err)

		response, err := http.DefaultClient.Do(request)

		require.NoError(t, err)
		_ = response.Body.Close()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
	})

	t.Run("closes the connection on an oversized client message", func(t *testing.T) {
		t.Parallel()

		hub, server := startHub(t, nil)
		conn := dial(t, server)
		broadcastUntilReceived(t, hub, conn, "ready")

		require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(strings.Repeat("x", 1024))))

		require.NoError(t, conn.SetReadDeadline(time.Now().Add(readTimeout)))
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				tooBig := websocket.IsCloseError(err, websocket.CloseMessageTooBig)
				assert.True(t, tooBig || !websocket.IsUnexpectedCloseError(err), "connection closed: %v", err)
				return
			}
		}
	})

	t.Run("disconnects a silent client", func(t *testing.T) {
		t.Parallel()

		hub, server := startHub(t, nil, ws.WithPongWait(200*time.Millisecond))
		conn := dial(t, server)
		// Answering pings is the client's job, so a client that never reads, never answers them
		conn.SetPingHandler(func(string) error { return nil })
		broadcastUntilReceived(t, hub, conn, "ready")

		require.NoError(t, conn.SetReadDeadline(time.Now().Add(readTimeout)))
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				netErr, ok := errors.AsType[net.Error](err)
				assert.False(t, ok && netErr.Timeout(), "the hub closed the connection: %v", err)
				return
			}
		}
	})

	t.Run("keeps a client connected that answers the pings", func(t *testing.T) {
		t.Parallel()

		hub, server := startHub(t, nil, ws.WithPongWait(200*time.Millisecond))
		conn := dial(t, server)
		pings := make(chan struct{}, 10)
		conn.SetPingHandler(func(data string) error {
			select {
			case pings <- struct{}{}:
			default:
			}
			return conn.WriteControl(websocket.PongMessage, []byte(data), time.Now().Add(time.Second))
		})
		broadcastUntilReceived(t, hub, conn, "ready")

		// Reading processes the pings, and the answers keep the connection open past several pong waits
		go func() {
			time.Sleep(time.Second)
			hub.Broadcast([]byte("still connected"))
		}()

		assert.Equal(t, "still connected", readUntil(t, conn, "still connected"))
		assert.NotEmpty(t, pings)
	})

	t.Run("closes new connections on a stopped hub", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(stoppedHub(t))
		defer server.Close()

		conn, response, err := websocket.DefaultDialer.Dial(wsURL(server, ""), nil)

		require.NoError(t, err)
		_ = response.Body.Close()
		defer func() { _ = conn.Close() }()
		require.NoError(t, conn.SetReadDeadline(time.Now().Add(readTimeout)))
		_, _, err = conn.ReadMessage()
		assert.Error(t, err)
	})
}

func TestHub_Serve(t *testing.T) {
	t.Parallel()

	t.Run("registers the client with every topic", func(t *testing.T) {
		t.Parallel()

		hub, server := startTopicHub(t)
		conn := dialTopics(t, server, "news", "sports")
		broadcastUntilReceived(t, hub, conn, "ready")

		hub.Publish("news", []byte("headline"))
		hub.Publish("sports", []byte("score"))

		assert.Equal(t, "headline", readUntil(t, conn, "headline"))
		assert.Equal(t, "score", readUntil(t, conn, "score"))
	})
}

// stoppedHub returns a hub whose Run has already returned.
func stoppedHub(t *testing.T) *ws.Hub {
	t.Helper()

	hub := ws.New(zerolog.Nop(), nil)
	ctx, cancel := context.WithCancel(t.Context())
	stopped := make(chan struct{})
	go func() {
		hub.Run(ctx)
		close(stopped)
	}()
	cancel()
	<-stopped
	return hub
}

// startHub runs a hub behind a test server until the test ends.
func startHub(t *testing.T, snapshot ws.SnapshotFunc, opts ...ws.Option) (*ws.Hub, *httptest.Server) {
	t.Helper()

	hub := ws.New(zerolog.Nop(), snapshot, opts...)
	ctx, cancel := context.WithCancel(t.Context())
	go hub.Run(ctx)

	server := httptest.NewServer(hub)
	t.Cleanup(func() {
		server.Close()
		cancel()
	})
	return hub, server
}

// startTopicHub runs a hub behind a test server until the test ends. The server registers a client
// with the topics of its query.
func startTopicHub(t *testing.T) (*ws.Hub, *httptest.Server) {
	t.Helper()

	hub := ws.New(zerolog.Nop(), nil)
	ctx, cancel := context.WithCancel(t.Context())
	go hub.Run(ctx)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.Serve(w, r, r.URL.Query()["topic"])
	}))
	t.Cleanup(func() {
		server.Close()
		cancel()
	})
	return hub, server
}

// dial connects a client to the test server.
func dial(t *testing.T, server *httptest.Server) *websocket.Conn {
	t.Helper()

	return connect(t, wsURL(server, ""), nil)
}

// dialFrom connects a client to the test server with the Origin header of a page.
func dialFrom(t *testing.T, server *httptest.Server, origin string) *websocket.Conn {
	t.Helper()

	header := http.Header{}
	header.Set("Origin", origin)
	return connect(t, wsURL(server, ""), header)
}

// dialTopics connects a client of the topics to a server of startTopicHub.
func dialTopics(t *testing.T, server *httptest.Server, topics ...string) *websocket.Conn {
	t.Helper()

	query := url.Values{
		"topic": topics,
	}
	return connect(t, wsURL(server, "?"+query.Encode()), nil)
}

// connect dials the URL and closes the connection when the test ends.
func connect(t *testing.T, target string, header http.Header) *websocket.Conn {
	t.Helper()

	conn, response, err := websocket.DefaultDialer.Dial(target, header)
	require.NoError(t, err)
	_ = response.Body.Close()
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// wsURL returns the WebSocket URL of the test server with the query.
func wsURL(server *httptest.Server, query string) string {
	return "ws" + strings.TrimPrefix(server.URL, "http") + query
}

// allowOrigin returns an origin check that accepts only the origin.
func allowOrigin(origin string) func(r *http.Request) bool {
	return func(r *http.Request) bool {
		return r.Header.Get("Origin") == origin
	}
}

// broadcastUntilReceived broadcasts until the client receives the message. A freshly dialed client is registered
// asynchronously, so a message broadcast right after Dial may be sent before it is known. The broadcasts run
// in their own goroutine, since a read that times out leaves a gorilla connection unusable, so the client reads
// with the full timeout. Later reads skip leftover copies with readUntil.
func broadcastUntilReceived(t *testing.T, hub *ws.Hub, conn *websocket.Conn, message string) {
	t.Helper()

	stop := make(chan struct{})
	defer close(stop)
	go func() {
		ticker := time.NewTicker(20 * time.Millisecond)
		defer ticker.Stop()
		for {
			hub.Broadcast([]byte(message))
			select {
			case <-stop:
				return
			case <-ticker.C:
			}
		}
	}()

	readUntil(t, conn, message)
}

// readUntil skips messages until the wanted one arrives.
func readUntil(t *testing.T, conn *websocket.Conn, want string) string {
	t.Helper()

	for {
		if message := read(t, conn); message == want {
			return message
		}
	}
}

// readAll returns the messages up to the last one, which ends the reading.
func readAll(t *testing.T, conn *websocket.Conn, last string) []string {
	t.Helper()

	var messages []string
	for {
		message := read(t, conn)
		if message == last {
			return messages
		}
		messages = append(messages, message)
	}
}

// assertClosed checks that the hub closes the connection.
func assertClosed(t *testing.T, conn *websocket.Conn) {
	t.Helper()

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(readTimeout)))
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			closed := websocket.IsCloseError(err, websocket.CloseNoStatusReceived, websocket.CloseNormalClosure)
			assert.True(t, closed, "expected a close message, got %v", err)
			return
		}
	}
}

// read returns the next text message, failing the test after readTimeout.
func read(t *testing.T, conn *websocket.Conn) string {
	t.Helper()

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(readTimeout)))
	messageType, data, err := conn.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, websocket.TextMessage, messageType)
	return string(data)
}
