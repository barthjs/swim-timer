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

	"github.com/barthjs/swim-timer/internal/config"
)

// webhookWaitTimeout bounds how long a test waits for a webhook message.
const webhookWaitTimeout = 5 * time.Second

// withWebhookReceivers allows the loopback network, so webhooks can be delivered to the test receivers.
func withWebhookReceivers() option {
	return withConfig(func(cfg *config.Config) {
		cfg.Webhook.RawAllowedNetworks = "127.0.0.0/8,::1/128"
	})
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

// webhookReceiver is an endpoint that records the messages it receives.
// It answers with the given statuses in turn and repeats the last one, 204 without any.
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
