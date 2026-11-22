package domain

import "time"

// MessageType names what a websocket message announces.
type MessageType string

const (
	// MessageHello greets a client that just connected with the version and the settings of the application.
	MessageHello MessageType = "hello"
	// MessageHealth carries the state of every dependency, on connect and whenever one of them changes.
	MessageHealth MessageType = "health"
	// MessageChanged announces that a resource changed. It carries only its name, open pages fetch it again.
	MessageChanged MessageType = "changed"
	// MessageInit carries the live heat to an overlay that just connected.
	MessageInit MessageType = "init"
	// MessageUpdate carries a changed live heat.
	MessageUpdate MessageType = "update"
)

// Resource names what a changed message announces.
type Resource string

const (
	ResourceMeet   Resource = "meet"
	ResourcePreset Resource = "preset"
	ResourceTiming Resource = "timing"
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

// Hello is the data of the hello message. Meta changes with every setting the web interface reads,
// so a client notices a restart that applied changed settings.
type Hello struct {
	Version string `json:"version"`
	Meta    string `json:"meta"`
}

// Changed is the data of the changed message. ID names the changed item, empty when the whole resource changed.
type Changed struct {
	Resource Resource `json:"resource"`
	ID       string   `json:"id,omitzero"`
}

// NewMessage returns a message of the given type, stamped with the current time of the application.
func NewMessage(typ MessageType, data any) Message {
	return Message{
		Type:       typ,
		ServerTime: time.Now(),
		Data:       data,
	}
}
