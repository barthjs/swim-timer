package service

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
	"github.com/barthjs/swim-timer/internal/apperr"
	"github.com/barthjs/swim-timer/internal/domain"
	"github.com/barthjs/swim-timer/internal/media"
	"github.com/barthjs/swim-timer/internal/store"
	"github.com/barthjs/swim-timer/internal/webhook"
)

// webhookWaitTimeout bounds how long a test waits for a delivery in the background.
const webhookWaitTimeout = 5 * time.Second

func TestWebhookService_List(t *testing.T) {
	t.Parallel()

	t.Run("returns the webhooks with their state sorted by name", func(t *testing.T) {
		t.Parallel()
		svc := newTestWebhookService(t)
		receiver := newWebhookReceiver(t)
		second := createTestWebhook(t, svc, receiver.URL(), "b")
		first := createTestWebhook(t, svc, receiver.URL(), "a")
		_, err := svc.Test(t.Context(), first.Webhook.ID)
		require.NoError(t, err)

		states, err := svc.List(t.Context())

		require.NoError(t, err)
		require.Len(t, states, 2)
		assert.Equal(t, first.Webhook, states[0].Webhook)
		assert.False(t, states[0].Status.LastDeliveredAt.IsZero(), "the state of the test delivery is listed")
		assert.Equal(t, second.Webhook, states[1].Webhook)
	})

	t.Run("leaves out a webhook whose file cannot be read", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		storage, err := media.New(media.Options{
			Driver:    media.DriverLocal,
			LocalPath: root,
		})
		require.NoError(t, err)
		svc := newTestWebhookServiceWithStore(t, store.NewWebhookStore(storage), webhook.Options{
			AllowedNetworks: loopbackNetworks,
		})
		created := createTestWebhook(t, svc, newWebhookReceiver(t).URL(), apitest.Faker(t).Word())
		require.NoError(t, os.WriteFile(filepath.Join(root, "webhooks", "broken.json"), []byte("{not json"), 0o644))

		states, err := svc.List(t.Context())

		require.NoError(t, err)
		require.Len(t, states, 1)
		assert.Equal(t, created.Webhook, states[0].Webhook)
	})

	t.Run("returns no webhooks without any", func(t *testing.T) {
		t.Parallel()
		svc := newTestWebhookService(t)

		states, err := svc.List(t.Context())

		require.NoError(t, err)
		assert.Empty(t, states)
	})
}

func TestWebhookService_ListDeliveries(t *testing.T) {
	t.Parallel()

	t.Run("returns the deliveries, the most recent first", func(t *testing.T) {
		t.Parallel()
		svc := newTestWebhookService(t)
		created := createTestWebhook(t, svc, newWebhookReceiver(t).URL(), apitest.Faker(t).Word())
		first, err := svc.Test(t.Context(), created.Webhook.ID)
		require.NoError(t, err)
		second, err := svc.Test(t.Context(), created.Webhook.ID)
		require.NoError(t, err)

		deliveries, err := svc.ListDeliveries(t.Context(), created.Webhook.ID)

		require.NoError(t, err)
		assert.Equal(t, []WebhookDelivery{
			second,
			first,
		}, deliveries)
	})

	t.Run("keeps only the last deliveries", func(t *testing.T) {
		t.Parallel()
		svc := newTestWebhookService(t)
		created := createTestWebhook(t, svc, newWebhookReceiver(t).URL(), apitest.Faker(t).Word())
		for range webhookDeliveryLogSize + 1 {
			_, err := svc.Test(t.Context(), created.Webhook.ID)
			require.NoError(t, err)
		}

		deliveries, err := svc.ListDeliveries(t.Context(), created.Webhook.ID)

		require.NoError(t, err)
		assert.Len(t, deliveries, webhookDeliveryLogSize)
	})

	t.Run("returns no deliveries for a webhook without any", func(t *testing.T) {
		t.Parallel()
		svc := newTestWebhookService(t)
		receiver := newWebhookReceiver(t)
		wh := domain.Webhook{
			ID:      "stored",
			Name:    apitest.Faker(t).Word(),
			URL:     receiver.URL(),
			Enabled: true,
			Secret:  webhook.GenerateSecret(),
		}
		require.NoError(t, svc.store.Save(t.Context(), wh.ID, wh))

		deliveries, err := svc.ListDeliveries(t.Context(), wh.ID)

		require.NoError(t, err)
		assert.NotNil(t, deliveries)
		assert.Empty(t, deliveries)
	})

	t.Run("returns not found for an unknown webhook", func(t *testing.T) {
		t.Parallel()
		svc := newTestWebhookService(t)

		_, err := svc.ListDeliveries(t.Context(), "missing")

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})
}

func TestWebhookService_Get(t *testing.T) {
	t.Parallel()

	t.Run("returns the webhook", func(t *testing.T) {
		t.Parallel()
		svc := newTestWebhookService(t)
		created := createTestWebhook(t, svc, newWebhookReceiver(t).URL(), apitest.Faker(t).Word())

		state, err := svc.Get(t.Context(), created.Webhook.ID)

		require.NoError(t, err)
		assert.Equal(t, created, state)
	})

	t.Run("returns not found for an unknown webhook", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			id   string
		}{
			{
				name: "unknown id",
				id:   "missing",
			},
			{
				name: "id outside the folder",
				id:   "../missing",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				svc := newTestWebhookService(t)

				_, err := svc.Get(t.Context(), tt.id)

				assert.ErrorIs(t, err, apperr.ErrNotFound)
			})
		}
	})
}

func TestWebhookService_Create(t *testing.T) {
	t.Parallel()

	t.Run("stores the webhook under a generated id", func(t *testing.T) {
		t.Parallel()
		storage := newTestStorage(t)
		svc := newTestWebhookServiceWithStore(t, store.NewWebhookStore(storage), webhook.Options{
			AllowedNetworks: loopbackNetworks,
		})
		input := newWebhookCreateInput(t, newWebhookReceiver(t).URL())

		state, err := svc.Create(t.Context(), input)

		require.NoError(t, err)
		assert.NotEmpty(t, state.Webhook.ID)
		assert.Equal(t, input.Name, state.Webhook.Name)
		assert.Equal(t, input.URL, state.Webhook.URL)
		assert.Equal(t, input.Events, state.Webhook.Events)
		assert.True(t, state.Webhook.Enabled)
		assert.Equal(t, input.Secret, state.Webhook.Secret)
		stored, err := store.NewWebhookStore(storage).Get(t.Context(), state.Webhook.ID)
		require.NoError(t, err)
		assert.Equal(t, state.Webhook, *stored)
	})

	t.Run("generates a missing secret", func(t *testing.T) {
		t.Parallel()
		svc := newTestWebhookService(t)
		input := newWebhookCreateInput(t, newWebhookReceiver(t).URL())
		input.Secret = ""

		state, err := svc.Create(t.Context(), input)

		require.NoError(t, err)
		_, err = webhook.ParseSecret(state.Webhook.Secret)
		assert.NoError(t, err)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name  string
			url   string
			field string
			tag   string
		}{
			{
				name:  "malformed url",
				url:   "ftp://example.com/hook",
				field: "url",
				tag:   "url",
			},
			{
				name:  "url in a blocked network",
				url:   "https://192.168.1.10/hook",
				field: "url",
				tag:   "blocked_address",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				svc := newTestWebhookServiceWithStore(t, store.NewWebhookStore(newTestStorage(t)), webhook.Options{})

				_, err := svc.Create(t.Context(), newWebhookCreateInput(t, tt.url))

				assertValidationError(t, err, tt.field, tt.tag)
			})
		}
	})

	t.Run("rejects a malformed secret", func(t *testing.T) {
		t.Parallel()
		svc := newTestWebhookService(t)
		input := newWebhookCreateInput(t, newWebhookReceiver(t).URL())
		input.Secret = apitest.Faker(t).Word()

		_, err := svc.Create(t.Context(), input)

		assertValidationError(t, err, "secret", "webhook_secret")
	})
}

func TestWebhookService_Update(t *testing.T) {
	t.Parallel()

	t.Run("replaces the settings of the webhook", func(t *testing.T) {
		t.Parallel()
		svc := newTestWebhookService(t)
		created := createTestWebhook(t, svc, newWebhookReceiver(t).URL(), apitest.Faker(t).Word())
		input := WebhookUpdateInput{
			Name: apitest.Faker(t).Word(),
			URL:  newWebhookReceiver(t).URL(),
			Events: []domain.WebhookEvent{
				domain.WebhookEventMeetReloaded,
			},
			Enabled: false,
			Secret:  webhook.GenerateSecret(),
		}

		state, err := svc.Update(t.Context(), created.Webhook.ID, input)

		require.NoError(t, err)
		assert.Equal(t, domain.Webhook{
			ID:      created.Webhook.ID,
			Name:    input.Name,
			URL:     input.URL,
			Events:  input.Events,
			Enabled: false,
			Secret:  input.Secret,
		}, state.Webhook)
		stored, err := svc.Get(t.Context(), created.Webhook.ID)
		require.NoError(t, err)
		assert.Equal(t, state.Webhook, stored.Webhook)
	})

	t.Run("generates a new secret for an empty one", func(t *testing.T) {
		t.Parallel()
		svc := newTestWebhookService(t)
		created := createTestWebhook(t, svc, newWebhookReceiver(t).URL(), apitest.Faker(t).Word())
		input := WebhookUpdateInput(newWebhookCreateInput(t, created.Webhook.URL))
		input.Secret = ""

		state, err := svc.Update(t.Context(), created.Webhook.ID, input)

		require.NoError(t, err)
		assert.NotEqual(t, created.Webhook.Secret, state.Webhook.Secret)
		_, err = webhook.ParseSecret(state.Webhook.Secret)
		assert.NoError(t, err)
	})

	t.Run("rejects a malformed secret", func(t *testing.T) {
		t.Parallel()
		svc := newTestWebhookService(t)
		created := createTestWebhook(t, svc, newWebhookReceiver(t).URL(), apitest.Faker(t).Word())
		input := WebhookUpdateInput(newWebhookCreateInput(t, created.Webhook.URL))
		input.Secret = apitest.Faker(t).Word()

		_, err := svc.Update(t.Context(), created.Webhook.ID, input)

		assertValidationError(t, err, "secret", "webhook_secret")
	})

	t.Run("returns not found for an unknown webhook", func(t *testing.T) {
		t.Parallel()
		svc := newTestWebhookService(t)

		_, err := svc.Update(t.Context(), "missing", WebhookUpdateInput(newWebhookCreateInput(t, newWebhookReceiver(t).URL())))

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})
}

func TestWebhookService_Delete(t *testing.T) {
	t.Parallel()

	t.Run("removes the webhook and ends its deliveries", func(t *testing.T) {
		t.Parallel()
		svc := newTestWebhookService(t)
		receiver := newWebhookReceiver(t)
		created := createTestWebhook(t, svc, receiver.URL(), apitest.Faker(t).Word())
		runTestWebhookService(t, svc)

		err := svc.Delete(t.Context(), created.Webhook.ID)

		require.NoError(t, err)
		_, err = svc.Get(t.Context(), created.Webhook.ID)
		assert.ErrorIs(t, err, apperr.ErrNotFound)
		svc.Publish(domain.WebhookEventHeatUpdated, sampleData(nil))
		receiver.assertNone(t)
	})

	t.Run("returns not found for an unknown webhook", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			id   string
		}{
			{
				name: "unknown id",
				id:   "missing",
			},
			{
				name: "id outside the folder",
				id:   "../missing",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				svc := newTestWebhookService(t)

				err := svc.Delete(t.Context(), tt.id)

				assert.ErrorIs(t, err, apperr.ErrNotFound)
			})
		}
	})
}

func TestWebhookService_Test(t *testing.T) {
	t.Parallel()

	t.Run("delivers a test message", func(t *testing.T) {
		t.Parallel()
		svc := newTestWebhookService(t)
		receiver := newWebhookReceiver(t)
		created := createTestWebhook(t, svc, receiver.URL(), apitest.Faker(t).Word())

		delivery, err := svc.Test(t.Context(), created.Webhook.ID)

		require.NoError(t, err)
		assert.Equal(t, WebhookDeliveryDelivered, delivery.Outcome)
		assert.Equal(t, http.StatusNoContent, delivery.StatusCode)
		assert.Equal(t, webhookTestEvent, delivery.Event)
		message := receiver.next(t)
		assert.Equal(t, webhookTestEvent, message.Type)
		assert.Equal(t, delivery.MessageID, message.ID)
		state, err := svc.Get(t.Context(), created.Webhook.ID)
		require.NoError(t, err)
		assert.Equal(t, delivery.AttemptedAt, state.Status.LastDeliveredAt)
	})

	t.Run("accepts a self-signed certificate if the webhook skips the verification", func(t *testing.T) {
		t.Parallel()
		svc := newTestWebhookService(t)
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		t.Cleanup(server.Close)
		input := newWebhookCreateInput(t, server.URL)
		input.SkipTLSVerify = true
		created, err := svc.Create(t.Context(), input)
		require.NoError(t, err)

		delivery, err := svc.Test(t.Context(), created.Webhook.ID)

		require.NoError(t, err)
		assert.Equal(t, WebhookDeliveryDelivered, delivery.Outcome)
	})

	t.Run("tests a disabled webhook", func(t *testing.T) {
		t.Parallel()
		svc := newTestWebhookService(t)
		receiver := newWebhookReceiver(t)
		input := newWebhookCreateInput(t, receiver.URL())
		input.Enabled = false
		created, err := svc.Create(t.Context(), input)
		require.NoError(t, err)

		delivery, err := svc.Test(t.Context(), created.Webhook.ID)

		require.NoError(t, err)
		assert.Equal(t, WebhookDeliveryDelivered, delivery.Outcome)
	})

	t.Run("records a failed delivery in the state", func(t *testing.T) {
		t.Parallel()
		svc := newTestWebhookService(t)
		receiver := newWebhookReceiver(t, http.StatusInternalServerError)
		created := createTestWebhook(t, svc, receiver.URL(), apitest.Faker(t).Word())

		delivery, err := svc.Test(t.Context(), created.Webhook.ID)

		require.NoError(t, err)
		assert.Equal(t, WebhookDeliveryRetry, delivery.Outcome)
		state, err := svc.Get(t.Context(), created.Webhook.ID)
		require.NoError(t, err)
		assert.Equal(t, WebhookStatus{
			LastFailedAt:        delivery.AttemptedAt,
			LastStatusCode:      http.StatusInternalServerError,
			ConsecutiveFailures: 1,
		}, state.Status)
	})

	t.Run("names why a delivery got no response", func(t *testing.T) {
		t.Parallel()
		svc := newTestWebhookServiceWithStore(t, store.NewWebhookStore(newTestStorage(t)), webhook.Options{})
		receiver := newWebhookReceiver(t)
		// The address check of a save would reject the receiver, which the delivery has to find itself
		wh := domain.Webhook{
			ID:      "stored",
			Name:    apitest.Faker(t).Word(),
			URL:     receiver.URL(),
			Enabled: true,
			Secret:  webhook.GenerateSecret(),
		}
		require.NoError(t, svc.store.Save(t.Context(), wh.ID, wh))

		delivery, err := svc.Test(t.Context(), wh.ID)

		require.NoError(t, err)
		assert.Equal(t, WebhookDeliveryErrorBlockedAddress, delivery.ErrorCode)
		assert.NotEmpty(t, delivery.Error)
		state, err := svc.Get(t.Context(), wh.ID)
		require.NoError(t, err)
		assert.Equal(t, WebhookDeliveryErrorBlockedAddress, state.Status.LastErrorCode)
	})

	t.Run("gives a webhook deleted during the test no target", func(t *testing.T) {
		t.Parallel()
		svc := newTestWebhookService(t)
		var id string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// The admin deletes the webhook while its test is on its way
			assert.NoError(t, svc.Delete(r.Context(), id))
			w.WriteHeader(http.StatusNoContent)
		}))
		t.Cleanup(server.Close)
		id = createTestWebhook(t, svc, server.URL, apitest.Faker(t).Word()).Webhook.ID

		delivery, err := svc.Test(t.Context(), id)

		require.NoError(t, err)
		assert.Equal(t, WebhookDeliveryDelivered, delivery.Outcome)
		assert.Nil(t, svc.target(id), "the deleted webhook receives no further messages")
	})

	t.Run("returns not found for an unknown webhook", func(t *testing.T) {
		t.Parallel()
		svc := newTestWebhookService(t)

		_, err := svc.Test(t.Context(), "missing")

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})
}

func TestWebhookService_Preview(t *testing.T) {
	t.Parallel()

	t.Run("returns the signed message with the current data of the event", func(t *testing.T) {
		t.Parallel()
		svc := newTestWebhookService(t)
		receiver := newWebhookReceiver(t)
		created := createTestWebhook(t, svc, receiver.URL(), apitest.Faker(t).Word())

		request, err := svc.Preview(t.Context(), created.Webhook.ID, domain.WebhookEventMeetReloaded)

		require.NoError(t, err)
		assert.Equal(t, receiver.URL(), request.URL)
		assert.NotEmpty(t, request.Header.Get("webhook-signature"))
		var message receivedWebhook
		require.NoError(t, json.Unmarshal(request.Body, &message))
		assert.Equal(t, string(domain.WebhookEventMeetReloaded), message.Type)
		assert.JSONEq(t, `{"sample":"meet.reloaded"}`, string(message.Data))
		receiver.assertNone(t)
	})

	t.Run("returns not found for an unknown webhook", func(t *testing.T) {
		t.Parallel()
		svc := newTestWebhookService(t)

		_, err := svc.Preview(t.Context(), "missing", domain.WebhookEventHeatUpdated)

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})
}

func TestWebhookService_Publish(t *testing.T) {
	t.Parallel()

	t.Run("delivers the event to every webhook that subscribes to it", func(t *testing.T) {
		t.Parallel()
		svc := newTestWebhookService(t)
		subscribed := newWebhookReceiver(t)
		other := newWebhookReceiver(t)
		createTestWebhook(t, svc, subscribed.URL(), apitest.Faker(t).Word())
		otherInput := newWebhookCreateInput(t, other.URL())
		otherInput.Events = []domain.WebhookEvent{
			domain.WebhookEventMeetReloaded,
		}
		_, err := svc.Create(t.Context(), otherInput)
		require.NoError(t, err)
		runTestWebhookService(t, svc)
		data := map[string]int{
			"heat": apitest.Faker(t).IntRange(1, 10),
		}

		svc.Publish(domain.WebhookEventHeatUpdated, sampleData(data))

		message := subscribed.next(t)
		assert.Equal(t, string(domain.WebhookEventHeatUpdated), message.Type)
		assert.JSONEq(t, mustJSON(t, data), string(message.Data))
		other.assertNone(t)
	})

	t.Run("skips a disabled webhook", func(t *testing.T) {
		t.Parallel()
		svc := newTestWebhookService(t)
		receiver := newWebhookReceiver(t)
		input := newWebhookCreateInput(t, receiver.URL())
		input.Enabled = false
		_, err := svc.Create(t.Context(), input)
		require.NoError(t, err)
		runTestWebhookService(t, svc)

		svc.Publish(domain.WebhookEventHeatUpdated, sampleData(nil))

		receiver.assertNone(t)
	})

	t.Run("delivers only the latest message of an event to a busy target", func(t *testing.T) {
		t.Parallel()
		svc := newTestWebhookService(t)
		receiver := newWebhookReceiver(t)
		release := receiver.holdFirst()
		createTestWebhook(t, svc, receiver.URL(), apitest.Faker(t).Word())
		runTestWebhookService(t, svc)
		svc.Publish(domain.WebhookEventHeatUpdated, sampleData(1))
		receiver.waitForRequests(t, 1)
		svc.Publish(domain.WebhookEventHeatUpdated, sampleData(2))

		svc.Publish(domain.WebhookEventHeatUpdated, sampleData(3))

		release()
		assert.Equal(t, "1", string(receiver.next(t).Data))
		assert.Equal(t, "3", string(receiver.next(t).Data), "the newer message replaced the waiting one")
		receiver.assertNone(t)
	})

	t.Run("retries a failed delivery with the latest message", func(t *testing.T) {
		t.Parallel()
		svc := newTestWebhookService(t)
		receiver := newWebhookReceiver(t, http.StatusInternalServerError, http.StatusNoContent)
		createTestWebhook(t, svc, receiver.URL(), apitest.Faker(t).Word())
		runTestWebhookService(t, svc)
		svc.Publish(domain.WebhookEventHeatUpdated, sampleData(1))
		assert.Equal(t, "1", string(receiver.next(t).Data))

		svc.Publish(domain.WebhookEventHeatUpdated, sampleData(2))

		assert.Equal(t, "2", string(receiver.next(t).Data), "the retry sends the newer message")
	})

	t.Run("keeps delivering the other events while one fails", func(t *testing.T) {
		t.Parallel()
		svc := newTestWebhookService(t)
		heats := make(chan string, 10)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var message receivedWebhook
			_ = json.UnmarshalRead(r.Body, &message)
			if message.Type == string(domain.WebhookEventMeetReloaded) {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			heats <- string(message.Data)
			w.WriteHeader(http.StatusNoContent)
		}))
		t.Cleanup(server.Close)
		input := newWebhookCreateInput(t, server.URL)
		input.Events = domain.WebhookEvents()
		_, err := svc.Create(t.Context(), input)
		require.NoError(t, err)
		runTestWebhookService(t, svc)
		svc.Publish(domain.WebhookEventMeetReloaded, sampleData(nil))

		svc.Publish(domain.WebhookEventHeatUpdated, sampleData(1))

		select {
		case data := <-heats:
			assert.Equal(t, "1", data)
		case <-time.After(webhookWaitTimeout):
			t.Fatal("the failing meet held back the live heat")
		}
	})

	t.Run("builds no data without a subscribed webhook", func(t *testing.T) {
		t.Parallel()
		svc := newTestWebhookService(t)
		runTestWebhookService(t, svc)
		built := false

		svc.Publish(domain.WebhookEventHeatUpdated, func() any {
			built = true
			return nil
		})

		assert.False(t, built)
	})

	t.Run("disables a webhook whose target is gone", func(t *testing.T) {
		t.Parallel()
		svc := newTestWebhookService(t)
		receiver := newWebhookReceiver(t, http.StatusGone)
		created := createTestWebhook(t, svc, receiver.URL(), apitest.Faker(t).Word())
		runTestWebhookService(t, svc)

		svc.Publish(domain.WebhookEventHeatUpdated, sampleData(nil))

		receiver.next(t)
		require.Eventually(t, func() bool {
			state, err := svc.Get(t.Context(), created.Webhook.ID)
			return err == nil && !state.Webhook.Enabled
		}, webhookWaitTimeout, 10*time.Millisecond)
	})

	t.Run("keeps a webhook whose url changed while its old target answered gone", func(t *testing.T) {
		t.Parallel()
		svc := newTestWebhookService(t)
		replacement := newWebhookReceiver(t)
		var id string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// The admin moves the webhook to another URL while the old target answers
			input := WebhookUpdateInput(newWebhookCreateInput(t, replacement.URL()))
			_, err := svc.Update(r.Context(), id, input)
			assert.NoError(t, err)
			w.WriteHeader(http.StatusGone)
		}))
		t.Cleanup(server.Close)
		id = createTestWebhook(t, svc, server.URL, apitest.Faker(t).Word()).Webhook.ID
		runTestWebhookService(t, svc)

		svc.Publish(domain.WebhookEventHeatUpdated, sampleData(nil))

		require.Eventually(t, func() bool {
			deliveries, err := svc.ListDeliveries(t.Context(), id)
			return err == nil && len(deliveries) > 0 && deliveries[0].Outcome == WebhookDeliveryGone
		}, webhookWaitTimeout, 10*time.Millisecond)
		assert.Never(t, func() bool {
			state, err := svc.Get(t.Context(), id)
			return err != nil || !state.Webhook.Enabled
		}, 300*time.Millisecond, 10*time.Millisecond, "the answer of the old URL does not apply to the new one")
	})
}

func TestWebhookService_Run(t *testing.T) {
	t.Parallel()

	t.Run("delivers to the stored webhooks", func(t *testing.T) {
		t.Parallel()
		svc := newTestWebhookService(t)
		receiver := newWebhookReceiver(t)
		wh := domain.Webhook{
			ID:   "stored",
			Name: apitest.Faker(t).Word(),
			URL:  receiver.URL(),
			Events: []domain.WebhookEvent{
				domain.WebhookEventMeetReloaded,
			},
			Enabled: true,
			Secret:  webhook.GenerateSecret(),
		}
		require.NoError(t, svc.store.Save(t.Context(), wh.ID, wh))

		runTestWebhookService(t, svc)

		svc.Publish(domain.WebhookEventMeetReloaded, sampleData(nil))
		assert.Equal(t, string(domain.WebhookEventMeetReloaded), receiver.next(t).Type)
	})

	t.Run("returns once ctx is canceled", func(t *testing.T) {
		t.Parallel()
		svc := newTestWebhookService(t)
		createTestWebhook(t, svc, newWebhookReceiver(t).URL(), apitest.Faker(t).Word())
		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan struct{})
		go func() {
			svc.Run(ctx)
			close(done)
		}()

		cancel()

		select {
		case <-done:
		case <-time.After(webhookWaitTimeout):
			t.Fatal("Run did not return")
		}
	})
}

// loopbackNetworks allow the test receivers, which listen on the loopback address.
var loopbackNetworks = []netip.Prefix{
	netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("::1/128"),
}

// newTestWebhookService returns a WebhookService on a fresh storage that may deliver to the loopback address.
func newTestWebhookService(t *testing.T) *WebhookService {
	t.Helper()

	return newTestWebhookServiceWithStore(t, store.NewWebhookStore(newTestStorage(t)), webhook.Options{
		AllowedNetworks: loopbackNetworks,
	})
}

// newTestWebhookServiceWithStore returns a WebhookService on the store with a client built from options.
func newTestWebhookServiceWithStore(t *testing.T, webhooks *store.WebhookStore, options webhook.Options) *WebhookService {
	t.Helper()

	return NewWebhookService(zerolog.Nop(), webhook.New(options), webhooks, func(event domain.WebhookEvent) any {
		return map[string]string{
			"sample": string(event),
		}
	})
}

// runTestWebhookService runs the service until the test ends and waits until it delivers.
func runTestWebhookService(t *testing.T, svc *WebhookService) {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		svc.Run(ctx)
		close(done)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})

	require.Eventually(t, func() bool {
		svc.mu.Lock()
		defer svc.mu.Unlock()
		return svc.runCtx != nil
	}, webhookWaitTimeout, time.Millisecond)
}

// createTestWebhook creates an enabled webhook for the heat updates with the given name at the URL.
func createTestWebhook(t *testing.T, svc *WebhookService, url, name string) WebhookState {
	t.Helper()

	input := newWebhookCreateInput(t, url)
	input.Name = name
	state, err := svc.Create(t.Context(), input)
	require.NoError(t, err)
	return state
}

// newWebhookCreateInput returns an enabled webhook for the heat updates at the URL with a random name and secret.
func newWebhookCreateInput(t *testing.T, url string) WebhookCreateInput {
	t.Helper()

	return WebhookCreateInput{
		Name: apitest.Faker(t).Word(),
		URL:  url,
		Events: []domain.WebhookEvent{
			domain.WebhookEventHeatUpdated,
		},
		Enabled: true,
		Secret:  webhook.GenerateSecret(),
	}
}

// assertValidationError checks that err is a validation error with the tag on the field.
func assertValidationError(t *testing.T, err error, field, tag string) {
	t.Helper()

	validationErr, ok := errors.AsType[*apperr.ValidationError](err)
	require.True(t, ok, "a validation error, got %v", err)
	require.NotEmpty(t, validationErr.Fields[field], "an error on %s", field)
	assert.Equal(t, tag, validationErr.Fields[field][0].Tag)
}

// mustJSON encodes v as JSON.
func mustJSON(t *testing.T, v any) string {
	t.Helper()

	data, err := json.Marshal(v)
	require.NoError(t, err)
	return string(data)
}

// receivedWebhook is a message a webhookReceiver received.
type receivedWebhook struct {
	ID   string         `json:"id"`
	Type string         `json:"type"`
	Data jsontext.Value `json:"data"`
}

// webhookReceiver is an endpoint that records the messages it receives. It answers with the given statuses
// in turn and repeats the last one, 204 without any.
type webhookReceiver struct {
	server   *httptest.Server
	messages chan receivedWebhook

	mu       sync.Mutex
	statuses []int
	requests int
	// hold blocks the first request until it is closed, nil without one.
	hold chan struct{}
}

// newWebhookReceiver starts a receiver that answers with the statuses in turn and stops when the test ends.
func newWebhookReceiver(t *testing.T, statuses ...int) *webhookReceiver {
	t.Helper()

	receiver := &webhookReceiver{
		messages: make(chan receivedWebhook, 100),
		statuses: statuses,
	}
	receiver.server = httptest.NewServer(http.HandlerFunc(receiver.handle))
	t.Cleanup(receiver.server.Close)
	return receiver
}

// URL returns the URL of the receiver.
func (r *webhookReceiver) URL() string {
	return r.server.URL
}

// holdFirst blocks the first request until the returned function is called.
func (r *webhookReceiver) holdFirst() func() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.hold = make(chan struct{})
	hold := r.hold
	return sync.OnceFunc(func() { close(hold) })
}

// handle records a message and answers with the next status.
func (r *webhookReceiver) handle(w http.ResponseWriter, req *http.Request) {
	var message receivedWebhook
	_ = json.UnmarshalRead(req.Body, &message)

	r.mu.Lock()
	r.requests++
	first := r.requests == 1
	hold := r.hold
	status := http.StatusNoContent
	if len(r.statuses) > 0 {
		status = r.statuses[min(r.requests, len(r.statuses))-1]
	}
	r.mu.Unlock()

	if first && hold != nil {
		<-hold
	}
	r.messages <- message
	w.WriteHeader(status)
}

// waitForRequests waits until the receiver got at least count requests.
func (r *webhookReceiver) waitForRequests(t *testing.T, count int) {
	t.Helper()

	require.Eventually(t, func() bool {
		r.mu.Lock()
		defer r.mu.Unlock()
		return r.requests >= count
	}, webhookWaitTimeout, time.Millisecond)
}

// next waits for the next received message.
func (r *webhookReceiver) next(t *testing.T) receivedWebhook {
	t.Helper()

	select {
	case message := <-r.messages:
		return message
	case <-time.After(webhookWaitTimeout):
		t.Fatal("no webhook message received")
		return receivedWebhook{}
	}
}

// assertNone checks that the receiver gets no further message within a short time.
func (r *webhookReceiver) assertNone(t *testing.T) {
	t.Helper()

	select {
	case message := <-r.messages:
		t.Fatalf("unexpected webhook message %s", message.Type)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestDeliveryErrorCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want WebhookDeliveryErrorCode
	}{
		{
			name: "blocked address",
			err:  fmt.Errorf("dial: %w", webhook.ErrBlockedAddress),
			want: WebhookDeliveryErrorBlockedAddress,
		},
		{
			name: "untrusted certificate",
			err:  fmt.Errorf("%w: %w", webhook.ErrUntrustedCertificate, errors.New("x509: certificate signed by unknown authority")),
			want: WebhookDeliveryErrorCertificate,
		},
		{
			name: "invalid url",
			err:  webhook.ErrInvalidURL,
			want: WebhookDeliveryErrorInvalidURL,
		},
		{
			name: "invalid secret",
			err:  webhook.ErrInvalidSecret,
			want: WebhookDeliveryErrorInvalidSecret,
		},
		{
			name: "deadline",
			err:  fmt.Errorf("post: %w", context.DeadlineExceeded),
			want: WebhookDeliveryErrorTimeout,
		},
		{
			name: "network timeout",
			err: &net.OpError{
				Op:  "dial",
				Err: os.ErrDeadlineExceeded,
			},
			want: WebhookDeliveryErrorTimeout,
		},
		{
			name: "refused connection",
			err:  errors.New("connection refused"),
			want: WebhookDeliveryErrorUnreachable,
		},
	}

	for _, tt := range tests {
		t.Run("names the "+tt.name, func(t *testing.T) {
			t.Parallel()

			got := deliveryErrorCode(tt.err)

			assert.Equal(t, tt.want, got)
		})
	}
}

// sampleData returns the data of a published message as Publish asks for it.
func sampleData(data any) func() any {
	return func() any { return data }
}
