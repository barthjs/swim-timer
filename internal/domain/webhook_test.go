package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWebhookEvents(t *testing.T) {
	t.Parallel()

	t.Run("returns a copy", func(t *testing.T) {
		t.Parallel()

		events := WebhookEvents()
		events[0] = "changed"

		assert.Equal(t, WebhookEventHeatUpdated, WebhookEvents()[0])
	})
}

func TestParseWebhookEvent(t *testing.T) {
	t.Parallel()

	t.Run("returns a known event", func(t *testing.T) {
		t.Parallel()

		event, ok := ParseWebhookEvent("meet.reloaded")

		assert.True(t, ok)
		assert.Equal(t, WebhookEventMeetReloaded, event)
	})

	t.Run("rejects an unknown event", func(t *testing.T) {
		t.Parallel()

		_, ok := ParseWebhookEvent("webhook.test")

		assert.False(t, ok)
	})
}

func TestWebhook_Subscribes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		webhook Webhook
		want    bool
	}{
		{
			name: "subscribes to an event of an enabled webhook",
			webhook: Webhook{
				Events: []WebhookEvent{
					WebhookEventHeatUpdated,
				},
				Enabled: true,
			},
			want: true,
		},
		{
			name: "ignores an event the webhook does not list",
			webhook: Webhook{
				Events: []WebhookEvent{
					WebhookEventMeetReloaded,
				},
				Enabled: true,
			},
			want: false,
		},
		{
			name: "ignores every event of a disabled webhook",
			webhook: Webhook{
				Events: []WebhookEvent{
					WebhookEventHeatUpdated,
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.webhook.Subscribes(WebhookEventHeatUpdated)

			assert.Equal(t, tt.want, got)
		})
	}
}
