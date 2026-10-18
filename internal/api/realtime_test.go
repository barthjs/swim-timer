package api_test

import (
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/domain"
	"github.com/barthjs/swim-timer/internal/omega"
)

// liveMessage is a message of the websocket the overlays receive.
type liveMessage struct {
	Type       string           `json:"type"`
	ServerTime time.Time        `json:"serverTime"`
	Data       *domain.LiveHeat `json:"data"`
}

// TestRealtimeAccess checks the status of the WebSocket route for each kind of client. Every client may connect,
// so a request without a WebSocket upgrade fails alike for all of them.
func TestRealtimeAccess(t *testing.T) {
	t.Parallel()

	testAccess(t, []accessRoute{
		{
			method: http.MethodGet,
			path:   "/api/ws",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusBadRequest,
				clientAdmin: http.StatusBadRequest,
			},
		},
	})
}

// TestRealtimeConnectHandler checks the WebSocket connection of an overlay over a real server.
func TestRealtimeConnectHandler(t *testing.T) {
	t.Parallel()

	t.Run("sends the live heat on connect", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		message := readMessage(t, h.connectOverlay())

		assert.Equal(t, "init", message.Type)
		assert.WithinDuration(t, time.Now(), message.ServerTime, eventuallyTimeout)
		require.NotNil(t, message.Data)
		assert.Empty(t, message.Data.Lanes)
	})

	t.Run("publishes every frame", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		meet := newMeetFixture(h.Faker, 1)
		h.loadMeet(meet)
		conn := h.connectOverlay()
		readMessage(t, conn)

		h.Device.Send(t, omega.FrameData{
			Kind:     omega.KindLoad,
			LapCount: h.lapCount(meet),
			Event:    meet.event.Number,
			Heat:     1,
		})

		message := readMessage(t, conn)
		assert.Equal(t, "update", message.Type)
		require.NotNil(t, message.Data)
		assert.Equal(t, domain.HeatStatusLoad, message.Data.Status)
		assert.Len(t, message.Data.Lanes, meetLaneCount)
	})

	t.Run("announces a reloaded meet", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		conn := h.connectOverlay()
		readMessage(t, conn)

		h.loadMeet(newMeetFixture(h.Faker, 1))

		message := readMessage(t, conn)
		assert.Equal(t, "meet", message.Type)
		assert.Nil(t, message.Data, "pages fetch the meet themselves")
	})
}

// connectOverlay opens the websocket like an overlay does. The websocket needs a real connection,
// so the application is served by a test server for this.
func (h *harness) connectOverlay() *websocket.Conn {
	h.t.Helper()

	srv := httptest.NewServer(h.handler)
	h.t.Cleanup(srv.Close)

	conn, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/api/ws", nil)
	require.NoError(h.t, err)
	_ = response.Body.Close()
	h.t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// readMessage reads the next websocket message, failing the test if none arrives in time.
func readMessage(t *testing.T, conn *websocket.Conn) liveMessage {
	t.Helper()

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(eventuallyTimeout)))
	_, data, err := conn.ReadMessage()
	require.NoError(t, err)

	var message liveMessage
	require.NoError(t, json.Unmarshal(data, &message))
	return message
}
