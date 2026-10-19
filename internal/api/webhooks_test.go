package api_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/api"
	"github.com/barthjs/swim-timer/internal/config"
	"github.com/barthjs/swim-timer/internal/omega"
)

func TestWebhookAccess(t *testing.T) {
	t.Parallel()

	testAccess(t, []accessRoute{
		{
			method: http.MethodGet,
			path:   "/api/webhooks",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodGet,
			path:   "/api/webhooks/{id}",
			arrange: func(h *harness) (string, any) {
				return webhookPath(h.createWebhook(newWebhookReceiver(h.t).URL())), nil
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodPost,
			path:   "/api/webhooks",
			arrange: func(h *harness) (string, any) {
				return "/api/webhooks", newWebhookCreateRequest(h.Faker, newWebhookReceiver(h.t).URL())
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusCreated,
			},
		},
		{
			method: http.MethodPut,
			path:   "/api/webhooks/{id}",
			arrange: func(h *harness) (string, any) {
				receiver := newWebhookReceiver(h.t)
				return webhookPath(h.createWebhook(receiver.URL())), api.WebhookUpdateRequest(newWebhookCreateRequest(h.Faker, receiver.URL()))
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodDelete,
			path:   "/api/webhooks/{id}",
			arrange: func(h *harness) (string, any) {
				return webhookPath(h.createWebhook(newWebhookReceiver(h.t).URL())), nil
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusNoContent,
			},
		},
		{
			method: http.MethodPost,
			path:   "/api/webhooks/{id}/test",
			arrange: func(h *harness) (string, any) {
				return webhookPath(h.createWebhook(newWebhookReceiver(h.t).URL())) + "/test", nil
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodGet,
			path:   "/api/webhooks/{id}/deliveries",
			arrange: func(h *harness) (string, any) {
				return webhookPath(h.createWebhook(newWebhookReceiver(h.t).URL())) + "/deliveries", nil
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodGet,
			path:   "/api/webhooks/{id}/preview",
			arrange: func(h *harness) (string, any) {
				return webhookPath(h.createWebhook(newWebhookReceiver(h.t).URL())) + "/preview?event=heat.updated", nil
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
			},
		},
	}, withWebhookReceivers())
}

func TestWebhookListHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the webhooks sorted by name", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withWebhookReceivers())
		receiver := newWebhookReceiver(t)
		admin := h.asAdmin()
		second := newWebhookCreateRequest(h.Faker, receiver.URL())
		second.Name = "b"
		first := newWebhookCreateRequest(h.Faker, receiver.URL())
		first.Name = "a"
		admin.POST("/api/webhooks", second).AssertCreated()
		admin.POST("/api/webhooks", first).AssertCreated()

		response := admin.GET("/api/webhooks")

		response.AssertOK().
			AssertJSONCount("", 2).
			AssertJSONPath("0.name", "a").
			AssertJSONPath("0.url", receiver.URL()).
			AssertJSONPath("0.events", []any{"heat.updated"}).
			AssertJSONPath("0.status.consecutiveFailures", 0).
			AssertJSONPath("1.name", "b")
	})

	t.Run("returns an empty list without webhooks", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().GET("/api/webhooks")

		response.AssertOK().AssertJSON([]any{})
	})
}

func TestWebhookGetHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the webhook", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withWebhookReceivers())
		receiver := newWebhookReceiver(t)
		admin := h.asAdmin()
		request := newWebhookCreateRequest(h.Faker, receiver.URL())
		created := admin.POST("/api/webhooks", request).AssertCreated().Decode[api.WebhookResponse]()

		response := admin.GET(webhookPath(created.ID))

		response.AssertOK().
			AssertJSONPath("id", created.ID).
			AssertJSONPath("name", request.Name).
			AssertJSONPath("url", request.URL).
			AssertJSONPath("enabled", true).
			AssertJSONPath("secret", request.Secret).
			AssertJSONPathMissing("status.lastDeliveredAt")
	})

	t.Run("returns not found for an unknown webhook", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().GET(webhookPath(h.Faker.LetterN(26)))

		response.AssertNotFound()
	})
}

func TestWebhookCreateHandler(t *testing.T) {
	t.Parallel()

	t.Run("creates the webhook", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withWebhookReceivers())
		admin := h.asAdmin()
		request := newWebhookCreateRequest(h.Faker, newWebhookReceiver(t).URL())

		response := admin.POST("/api/webhooks", request)

		created := response.AssertCreated().Decode[api.WebhookResponse]()
		assert.NotEmpty(t, created.ID)
		assert.Equal(t, request.Name, created.Name)
		assert.Equal(t, request.Events, created.Events)
		assert.Equal(t, request.SkipTLSVerify, created.SkipTLSVerify)
		admin.GET(webhookPath(created.ID)).AssertOK().AssertJSONPath("secret", request.Secret)
	})

	t.Run("generates a missing secret", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withWebhookReceivers())
		request := newWebhookCreateRequest(h.Faker, newWebhookReceiver(t).URL())
		request.Secret = ""

		response := h.asAdmin().POST("/api/webhooks", request)

		created := response.AssertCreated().Decode[api.WebhookResponse]()
		assert.True(t, strings.HasPrefix(created.Secret, "whsec_"))
	})

	t.Run("delivers the live heat to the webhook", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withWebhookReceivers())
		meet := newMeetFixture(h.Faker, 1)
		h.loadMeet(meet)
		receiver := newWebhookReceiver(t)
		created := h.asAdmin().POST("/api/webhooks", newWebhookCreateRequest(h.Faker, receiver.URL())).
			AssertCreated().Decode[api.WebhookResponse]()

		h.Device.Send(t, omega.FrameData{
			Kind:     omega.KindLoad,
			LapCount: h.lapCount(meet),
			Event:    meet.event.Number,
			Heat:     1,
		})

		message := receiver.next(t)
		assert.Equal(t, "heat.updated", message.Type)
		var heat struct {
			Event int `json:"event"`
			Heat  int `json:"heat"`
		}
		require.NoError(t, json.Unmarshal(message.Data, &heat))
		assert.Equal(t, meet.event.Number, heat.Event)
		assert.Equal(t, 1, heat.Heat)
		assert.True(t, message.verify(created.Secret), "the message is signed with the secret of the webhook")
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name   string
			change func(*api.WebhookCreateRequest, *gofakeit.Faker)
			field  string
			tag    string
		}{
			{
				name:   "missing name",
				change: func(r *api.WebhookCreateRequest, _ *gofakeit.Faker) { r.Name = "" },
				field:  "name",
				tag:    "required",
			},
			{
				name:   "name too long",
				change: func(r *api.WebhookCreateRequest, f *gofakeit.Faker) { r.Name = f.LetterN(101) },
				field:  "name",
				tag:    "max",
			},
			{
				name:   "missing url",
				change: func(r *api.WebhookCreateRequest, _ *gofakeit.Faker) { r.URL = "" },
				field:  "url",
				tag:    "required",
			},
			{
				name:   "malformed url",
				change: func(r *api.WebhookCreateRequest, _ *gofakeit.Faker) { r.URL = "ftp://example.com/hook" },
				field:  "url",
				tag:    "url",
			},
			{
				name:   "url in a blocked network",
				change: func(r *api.WebhookCreateRequest, _ *gofakeit.Faker) { r.URL = "https://192.168.1.10/hook" },
				field:  "url",
				tag:    "blocked_address",
			},
			{
				name:   "missing events",
				change: func(r *api.WebhookCreateRequest, _ *gofakeit.Faker) { r.Events = nil },
				field:  "events",
				tag:    "required",
			},
			{
				name: "unknown event",
				change: func(r *api.WebhookCreateRequest, f *gofakeit.Faker) {
					r.Events = []string{
						f.Word(),
					}
				},
				field: "events[0]",
				tag:   "oneof",
			},
			{
				name:   "malformed secret",
				change: func(r *api.WebhookCreateRequest, f *gofakeit.Faker) { r.Secret = f.Word() },
				field:  "secret",
				tag:    "webhook_secret",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				h := newHarness(t, withWebhookReceivers())
				request := newWebhookCreateRequest(h.Faker, newWebhookReceiver(t).URL())
				tt.change(&request, h.Faker)

				response := h.asAdmin().POST("/api/webhooks", request)

				assertFieldError(t, response, tt.field, tt.tag)
			})
		}
	})
}

func TestWebhookUpdateHandler(t *testing.T) {
	t.Parallel()

	t.Run("replaces the settings of the webhook", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withWebhookReceivers())
		admin := h.asAdmin()
		id := h.createWebhook(newWebhookReceiver(t).URL())
		request := api.WebhookUpdateRequest(newWebhookCreateRequest(h.Faker, newWebhookReceiver(t).URL()))
		request.Events = []string{
			"meet.reloaded",
		}
		request.Enabled = false

		response := admin.PUT(webhookPath(id), request)

		response.AssertOK().
			AssertJSONPath("id", id).
			AssertJSONPath("name", request.Name).
			AssertJSONPath("url", request.URL).
			AssertJSONPath("events", []any{"meet.reloaded"}).
			AssertJSONPath("enabled", false).
			AssertJSONPath("secret", request.Secret)
		admin.GET(webhookPath(id)).AssertOK().AssertJSONPath("name", request.Name)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name   string
			change func(*api.WebhookUpdateRequest, *gofakeit.Faker)
			field  string
			tag    string
		}{
			{
				name:   "missing name",
				change: func(r *api.WebhookUpdateRequest, _ *gofakeit.Faker) { r.Name = "" },
				field:  "name",
				tag:    "required",
			},
			{
				name:   "url in a blocked network",
				change: func(r *api.WebhookUpdateRequest, _ *gofakeit.Faker) { r.URL = "https://192.168.1.10/hook" },
				field:  "url",
				tag:    "blocked_address",
			},
			{
				name:   "missing events",
				change: func(r *api.WebhookUpdateRequest, _ *gofakeit.Faker) { r.Events = nil },
				field:  "events",
				tag:    "required",
			},
			{
				name:   "malformed secret",
				change: func(r *api.WebhookUpdateRequest, f *gofakeit.Faker) { r.Secret = f.Word() },
				field:  "secret",
				tag:    "webhook_secret",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				h := newHarness(t, withWebhookReceivers())
				receiver := newWebhookReceiver(t)
				id := h.createWebhook(receiver.URL())
				request := api.WebhookUpdateRequest(newWebhookCreateRequest(h.Faker, receiver.URL()))
				tt.change(&request, h.Faker)

				response := h.asAdmin().PUT(webhookPath(id), request)

				assertFieldError(t, response, tt.field, tt.tag)
			})
		}
	})

	t.Run("returns not found for an unknown webhook", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withWebhookReceivers())
		request := api.WebhookUpdateRequest(newWebhookCreateRequest(h.Faker, newWebhookReceiver(t).URL()))

		response := h.asAdmin().PUT(webhookPath(h.Faker.LetterN(26)), request)

		response.AssertNotFound()
	})
}

func TestWebhookDeleteHandler(t *testing.T) {
	t.Parallel()

	t.Run("deletes the webhook", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withWebhookReceivers())
		admin := h.asAdmin()
		id := h.createWebhook(newWebhookReceiver(t).URL())

		response := admin.DELETE(webhookPath(id))

		response.AssertNoContent()
		admin.GET(webhookPath(id)).AssertNotFound()
	})

	t.Run("returns not found for an unknown webhook", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().DELETE(webhookPath(h.Faker.LetterN(26)))

		response.AssertNotFound()
	})
}

func TestWebhookTestHandler(t *testing.T) {
	t.Parallel()

	t.Run("delivers a signed test message", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withWebhookReceivers())
		admin := h.asAdmin()
		receiver := newWebhookReceiver(t)
		created := admin.POST("/api/webhooks", newWebhookCreateRequest(h.Faker, receiver.URL())).
			AssertCreated().Decode[api.WebhookResponse]()

		response := admin.POST(webhookPath(created.ID)+"/test", nil)

		delivery := response.AssertOK().Decode[api.WebhookDeliveryResponse]()
		assert.Equal(t, "delivered", delivery.Outcome)
		assert.Equal(t, http.StatusNoContent, delivery.StatusCode)
		assert.Equal(t, "webhook.test", delivery.Event)
		message := receiver.next(t)
		assert.Equal(t, delivery.MessageID, message.ID)
		assert.True(t, message.verify(created.Secret), "the message is signed with the secret of the webhook")
		admin.GET(webhookPath(created.ID)).AssertOK().AssertJSONPath("status.consecutiveFailures", 0)
	})

	t.Run("reports a failed delivery", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withWebhookReceivers())
		admin := h.asAdmin()
		id := h.createWebhook(newWebhookReceiver(t, http.StatusBadRequest).URL())

		response := admin.POST(webhookPath(id)+"/test", nil)

		response.AssertOK().AssertJSONPath("outcome", "rejected").AssertJSONPath("statusCode", http.StatusBadRequest)
		admin.GET(webhookPath(id)).AssertOK().
			AssertJSONPath("status.consecutiveFailures", 1).
			AssertJSONPath("status.lastStatusCode", http.StatusBadRequest).
			AssertJSONPathMissing("status.lastErrorCode")
	})

	t.Run("returns not found for an unknown webhook", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().POST(webhookPath(h.Faker.LetterN(26))+"/test", nil)

		response.AssertNotFound()
	})

	t.Run("is rate limited", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withWebhookReceivers())
		admin := h.asAdmin()
		id := h.createWebhook(newWebhookReceiver(t).URL())
		for range 10 {
			admin.POST(webhookPath(id)+"/test", nil).AssertOK()
		}

		response := admin.POST(webhookPath(id)+"/test", nil)

		response.AssertTooManyRequests().AssertJSONPath("code", "rate_limited")
		assert.NotEmpty(t, response.Header.Get("Retry-After"))
	})
}

func TestWebhookDeliveryListHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the deliveries, the most recent first", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withWebhookReceivers())
		admin := h.asAdmin()
		id := h.createWebhook(newWebhookReceiver(t).URL())
		first := admin.POST(webhookPath(id)+"/test", nil).AssertOK().Decode[api.WebhookDeliveryResponse]()
		second := admin.POST(webhookPath(id)+"/test", nil).AssertOK().Decode[api.WebhookDeliveryResponse]()

		response := admin.GET(webhookPath(id) + "/deliveries")

		response.AssertOK().
			AssertJSONCount("", 2).
			AssertJSONPath("0.messageId", second.MessageID).
			AssertJSONPath("1.messageId", first.MessageID)
	})

	t.Run("returns not found for an unknown webhook", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().GET(webhookPath(h.Faker.LetterN(26)) + "/deliveries")

		response.AssertNotFound()
	})
}

func TestWebhookPreviewGetHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the signed message with the current data without sending it", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withWebhookReceivers())
		meet := newMeetFixture(h.Faker, 2)
		h.loadMeet(meet)
		admin := h.asAdmin()
		receiver := newWebhookReceiver(t)
		created := admin.POST("/api/webhooks", newWebhookCreateRequest(h.Faker, receiver.URL())).
			AssertCreated().Decode[api.WebhookResponse]()

		response := admin.GET(webhookPath(created.ID) + "/preview?event=meet.reloaded")

		preview := response.AssertOK().Decode[api.WebhookPreviewResponse]()
		assert.Equal(t, http.MethodPost, preview.Method)
		assert.Equal(t, receiver.URL(), preview.URL)
		message := receivedWebhook{
			body:   []byte(preview.Body),
			header: http.Header{},
		}
		for _, header := range preview.Headers {
			message.header.Set(header.Name, header.Value)
		}
		message.signature = message.header.Get("webhook-signature")
		require.NoError(t, json.Unmarshal(message.body, &message))
		assert.Equal(t, "meet.reloaded", message.Type)
		var sent api.MeetResponse
		require.NoError(t, json.Unmarshal(message.Data, &sent))
		assert.Len(t, sent.StartEntries, 2*meetLaneCount, "the meet as it is loaded now")
		assert.True(t, message.verify(created.Secret), "the message is signed with the secret of the webhook")
		select {
		case <-receiver.messages:
			t.Fatal("the preview sent a message")
		case <-time.After(100 * time.Millisecond):
		}
	})

	t.Run("rejects an invalid event", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name  string
			query string
		}{
			{
				name:  "missing",
				query: "",
			},
			{
				name:  "unknown",
				query: "?event=webhook.test",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				h := newHarness(t, withWebhookReceivers())
				id := h.createWebhook(newWebhookReceiver(t).URL())

				response := h.asAdmin().GET(webhookPath(id) + "/preview" + tt.query)

				response.AssertBadRequest()
			})
		}
	})

	t.Run("returns not found for an unknown webhook", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().GET(webhookPath(h.Faker.LetterN(26)) + "/preview?event=heat.updated")

		response.AssertNotFound()
	})
}

// webhookWaitTimeout bounds how long a test waits for a webhook message.
const webhookWaitTimeout = 5 * time.Second

// withWebhookReceivers allows the loopback network, so webhooks can be delivered to the test receivers.
func withWebhookReceivers() option {
	return withConfig(func(cfg *config.Config) {
		cfg.Webhook.RawAllowedNetworks = "127.0.0.0/8,::1/128"
	})
}

// webhookPath returns the API path of a webhook.
func webhookPath(id string) string {
	return "/api/webhooks/" + id
}

// createWebhook creates an enabled webhook for the heat updates at the URL and returns its id.
func (h *harness) createWebhook(url string) string {
	h.t.Helper()

	created := h.asAdmin().POST("/api/webhooks", newWebhookCreateRequest(h.Faker, url)).AssertCreated().Decode[api.WebhookResponse]()
	return created.ID
}

// newWebhookCreateRequest returns an enabled webhook for the heat updates at the URL with a random name and secret.
func newWebhookCreateRequest(f *gofakeit.Faker, url string) api.WebhookCreateRequest {
	return api.WebhookCreateRequest{
		Name: f.Word(),
		URL:  url,
		Events: []string{
			"heat.updated",
		},
		Enabled:       true,
		Secret:        "whsec_" + base64.StdEncoding.EncodeToString([]byte(f.LetterN(32))),
		SkipTLSVerify: f.Bool(),
	}
}

// receivedWebhook is a message a webhookReceiver received with the headers that sign it.
type receivedWebhook struct {
	ID   string         `json:"id"`
	Type string         `json:"type"`
	Data jsontext.Value `json:"data"`

	body      []byte
	header    http.Header
	signature string
}

// verify reports whether the message carries a valid signature with the secret, as a receiver checks it.
func (m receivedWebhook) verify(secret string) bool {
	key, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(secret, "whsec_"))
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(m.header.Get("webhook-id") + "." + m.header.Get("webhook-timestamp") + "." + string(m.body)))
	return m.signature == "v1,"+base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// webhookReceiver is an endpoint that records the messages it receives. It answers with the given statuses
// in turn and repeats the last one, 204 without any.
type webhookReceiver struct {
	server   *httptest.Server
	messages chan receivedWebhook

	mu       sync.Mutex
	statuses []int
	requests int
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

// handle records a message and answers with the next status.
func (r *webhookReceiver) handle(w http.ResponseWriter, req *http.Request) {
	body, _ := io.ReadAll(req.Body)
	message := receivedWebhook{
		body:      body,
		header:    req.Header.Clone(),
		signature: req.Header.Get("webhook-signature"),
	}
	_ = json.Unmarshal(body, &message)

	r.mu.Lock()
	r.requests++
	status := http.StatusNoContent
	if len(r.statuses) > 0 {
		status = r.statuses[min(r.requests, len(r.statuses))-1]
	}
	r.mu.Unlock()

	r.messages <- message
	w.WriteHeader(status)
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
