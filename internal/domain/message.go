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

// Message is the envelope of every websocket message, sent as it is.
type Message struct {
	Type MessageType `json:"type"`
	// ServerTime is the clock of the application when the message was built, on which the overlays run the heat timer,
	// so a client with a different clock shows the same time.
	ServerTime time.Time `json:"serverTime"`
	// Data is left out when a message carries none, and the init and update messages carry the live heat.
	Data any `json:"data,omitzero"`
}

// NewMessage returns a message of the given type, stamped with the current time of the application.
func NewMessage(typ MessageType, data any) Message {
	return Message{
		Type:       typ,
		ServerTime: time.Now(),
		Data:       data,
	}
}
