package ws

import "github.com/barthjs/swim-timer/internal/data"

type wsMessageType string

const (
	wsMessageTypeInit   wsMessageType = "init"
	wsMessageTypeUpdate wsMessageType = "update"
)

// wsMessage holds the race data sent to the client via websocket
type wsMessage struct {
	Type wsMessageType   `json:"type"`
	Data *data.RaceState `json:"data"`
}
