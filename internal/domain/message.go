package domain

import "time"

// MessageType names what a websocket message announces.
type MessageType string

const (
	// MessageInit carries the live heat to an overlay that just connected.
	MessageInit MessageType = "init"
	// MessageUpdate carries a changed live heat.
	MessageUpdate MessageType = "update"
	// MessageMeet announces that the meet data was reloaded. It carries no data, open pages fetch the meet overview again.
	MessageMeet MessageType = "meet"
)

// Message is the envelope of every websocket message. It carries JSON tags because it is sent as-is.
// ServerTime is the clock of the application when the message was built.
// The overlays use it to run the heat timer on the clock of the application,
// so a client on another machine with a different clock shows the same time.
// Data is left out when a message carries none. The init and update messages always carry the live heat.
type Message struct {
	Type       MessageType `json:"type"`
	ServerTime time.Time   `json:"serverTime"`
	Data       any         `json:"data,omitzero"`
}

// NewMessage returns a message of the given type, stamped with the current time of the application.
func NewMessage(typ MessageType, data any) Message {
	return Message{
		Type:       typ,
		ServerTime: time.Now(),
		Data:       data,
	}
}
