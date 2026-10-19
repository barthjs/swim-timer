package domain

import "slices"

// WebhookEvent names what a webhook message announces.
type WebhookEvent string

const (
	// WebhookEventHeatUpdated carries the live heat after every change.
	WebhookEventHeatUpdated WebhookEvent = "heat.updated"
	// WebhookEventMeetReloaded carries the whole meet after a reload.
	WebhookEventMeetReloaded WebhookEvent = "meet.reloaded"
)

// webhookEvents lists every event a webhook can subscribe to.
var webhookEvents = []WebhookEvent{
	WebhookEventHeatUpdated,
	WebhookEventMeetReloaded,
}

// WebhookEvents returns every event a webhook can subscribe to.
// The slice is a copy, so callers cannot change the list.
func WebhookEvents() []WebhookEvent {
	return slices.Clone(webhookEvents)
}

// ParseWebhookEvent reports the event with the exact given name.
func ParseWebhookEvent(name string) (WebhookEvent, bool) {
	for _, event := range webhookEvents {
		if string(event) == name {
			return event, true
		}
	}
	return "", false
}

// Webhook is an endpoint the application delivers the events it subscribes to, signed with its secret.
// It carries JSON tags because the webhook store persists it as JSON.
type Webhook struct {
	ID      string         `json:"id"`
	Name    string         `json:"name"`
	URL     string         `json:"url"`
	Events  []WebhookEvent `json:"events"`
	Enabled bool           `json:"enabled"`
	Secret  string         `json:"secret"`
	// SkipTLSVerify accepts any certificate of the endpoint, such as a self-signed one in a local network.
	SkipTLSVerify bool `json:"skipTlsVerify"`
}

// Subscribes reports whether the webhook is enabled and subscribes to the event.
func (w Webhook) Subscribes(event WebhookEvent) bool {
	return w.Enabled && slices.Contains(w.Events, event)
}
