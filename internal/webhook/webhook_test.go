package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loopbackNetworks allow the test servers, which listen on the loopback address.
var loopbackNetworks = []netip.Prefix{
	netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("::1/128"),
}

func TestGenerateSecret(t *testing.T) {
	t.Parallel()

	t.Run("returns a valid secret", func(t *testing.T) {
		t.Parallel()

		secret := GenerateSecret()

		key, err := ParseSecret(secret)
		require.NoError(t, err)
		assert.Len(t, key, generatedSecretBytes)
	})

	t.Run("returns a new secret every time", func(t *testing.T) {
		t.Parallel()

		first, second := GenerateSecret(), GenerateSecret()

		assert.NotEqual(t, first, second)
	})
}

func TestParseSecret(t *testing.T) {
	t.Parallel()

	t.Run("returns the key", func(t *testing.T) {
		t.Parallel()
		key := randomKey(generatedSecretBytes)

		got, err := ParseSecret(encodeSecret(key))

		require.NoError(t, err)
		assert.Equal(t, key, got)
	})

	t.Run("rejects an invalid secret", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name   string
			secret string
		}{
			{
				name:   "missing prefix",
				secret: base64.StdEncoding.EncodeToString(randomKey(generatedSecretBytes)),
			},
			{
				name:   "malformed base64",
				secret: secretPrefix + "not base64!",
			},
			{
				name:   "too short",
				secret: encodeSecret(randomKey(minSecretBytes - 1)),
			},
			{
				name:   "too long",
				secret: encodeSecret(randomKey(maxSecretBytes + 1)),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				_, err := ParseSecret(tt.secret)

				assert.ErrorIs(t, err, ErrInvalidSecret)
			})
		}
	})
}

func TestNewPayload(t *testing.T) {
	t.Parallel()

	t.Run("encodes the message", func(t *testing.T) {
		t.Parallel()
		data := map[string]int{
			"count": 4,
		}

		payload, err := NewPayload("item.updated", data)

		require.NoError(t, err)
		var decoded struct {
			ID        string         `json:"id"`
			Type      string         `json:"type"`
			CreatedAt time.Time      `json:"createdAt"`
			Data      map[string]int `json:"data"`
		}
		require.NoError(t, json.Unmarshal(payload.Body, &decoded))
		assert.Equal(t, payload.ID, decoded.ID)
		assert.True(t, strings.HasPrefix(payload.ID, messageIDPrefix))
		assert.Equal(t, "item.updated", decoded.Type)
		assert.WithinDuration(t, time.Now(), decoded.CreatedAt, time.Minute)
		assert.Equal(t, data, decoded.Data)
	})

	t.Run("uses a new id for every message", func(t *testing.T) {
		t.Parallel()

		first, err := NewPayload("item.updated", nil)
		require.NoError(t, err)
		second, err := NewPayload("item.updated", nil)
		require.NoError(t, err)

		assert.NotEqual(t, first.ID, second.ID)
	})

	t.Run("rejects data that cannot be encoded", func(t *testing.T) {
		t.Parallel()

		_, err := NewPayload("item.updated", make(chan int))

		assert.Error(t, err)
	})
}

func TestClient_CheckURL(t *testing.T) {
	t.Parallel()

	t.Run("accepts a url", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name    string
			options Options
			url     string
		}{
			{
				name:    "https",
				options: Options{},
				url:     "https://example.com/hook",
			},
			{
				name:    "http",
				options: Options{},
				url:     "http://example.com/hook",
			},
			{
				name:    "NAT64 address of a public address",
				options: Options{},
				url:     "https://[64:ff9b::808:808]/hook",
			},
			{
				name: "private address in an allowed network",
				options: Options{
					AllowedNetworks: []netip.Prefix{
						netip.MustParsePrefix("10.0.0.0/8"),
					},
				},
				url: "https://10.1.2.3/hook",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				client := New(tt.options)

				err := client.CheckURL(tt.url)

				assert.NoError(t, err)
			})
		}
	})

	t.Run("rejects an invalid url", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			url  string
		}{
			{
				name: "other scheme",
				url:  "ftp://example.com/hook",
			},
			{
				name: "missing host",
				url:  "https:///hook",
			},
			{
				name: "malformed",
				url:  "https://exa mple.com/hook",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				client := New(Options{})

				err := client.CheckURL(tt.url)

				assert.ErrorIs(t, err, ErrInvalidURL)
			})
		}
	})

	t.Run("rejects an address in a blocked network", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			url  string
		}{
			{
				name: "loopback",
				url:  "https://127.0.0.1/hook",
			},
			{
				name: "IPv6 loopback",
				url:  "https://[::1]/hook",
			},
			{
				name: "IPv4-mapped loopback",
				url:  "https://[::ffff:127.0.0.1]/hook",
			},
			{
				name: "private",
				url:  "https://192.168.1.10/hook",
			},
			{
				name: "IPv6 unique local",
				url:  "https://[fd00::1]/hook",
			},
			{
				name: "link-local",
				url:  "https://169.254.169.254/hook",
			},
			{
				name: "carrier-grade NAT",
				url:  "https://100.64.0.1/hook",
			},
			{
				name: "unspecified",
				url:  "https://0.0.0.0/hook",
			},
			{
				name: "multicast",
				url:  "https://224.0.0.1/hook",
			},
			{
				name: "NAT64 address of a private address",
				url:  "https://[64:ff9b::c0a8:10a]/hook",
			},
			{
				name: "local-use NAT64",
				url:  "https://[64:ff9b:1::1]/hook",
			},
			{
				name: "6to4 address of a private address",
				url:  "https://[2002:c0a8:10a::1]/hook",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				client := New(Options{})

				err := client.CheckURL(tt.url)

				assert.ErrorIs(t, err, ErrBlockedAddress)
			})
		}
	})
}

func TestClient_Send(t *testing.T) {
	t.Parallel()

	t.Run("delivers the payload with a signature", func(t *testing.T) {
		t.Parallel()
		var received *http.Request
		var receivedBody []byte
		server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			received = r
			receivedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusNoContent)
		})
		key := randomKey(generatedSecretBytes)
		payload := newTestPayload(t)
		client := New(Options{
			UserAgent:       "webhook-test",
			AllowedNetworks: loopbackNetworks,
		})

		result := client.Send(t.Context(), Target{
			URL: server.URL,
			Secrets: []string{
				encodeSecret(key),
			},
		}, payload)

		require.NoError(t, result.Err)
		assert.Equal(t, OutcomeDelivered, result.Outcome)
		assert.Equal(t, http.StatusNoContent, result.StatusCode)
		require.NotNil(t, received)
		assert.Equal(t, payload.Body, receivedBody)
		assert.Equal(t, "application/json", received.Header.Get("Content-Type"))
		assert.Equal(t, "webhook-test", received.Header.Get("User-Agent"))
		assert.Equal(t, payload.ID, received.Header.Get("webhook-id"))
		timestamp := received.Header.Get("webhook-timestamp")
		assert.Equal(t, expectedSignature(key, payload.ID, timestamp, payload.Body), received.Header.Get("webhook-signature"))
	})

	t.Run("signs with every secret", func(t *testing.T) {
		t.Parallel()
		var received *http.Request
		server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			received = r
			w.WriteHeader(http.StatusOK)
		})
		first, second := randomKey(generatedSecretBytes), randomKey(generatedSecretBytes)
		payload := newTestPayload(t)
		client := New(Options{
			AllowedNetworks: loopbackNetworks,
		})

		result := client.Send(t.Context(), Target{
			URL: server.URL,
			Secrets: []string{
				encodeSecret(first),
				encodeSecret(second),
			},
		}, payload)

		require.Equal(t, OutcomeDelivered, result.Outcome)
		timestamp := received.Header.Get("webhook-timestamp")
		assert.Equal(t, []string{
			expectedSignature(first, payload.ID, timestamp, payload.Body),
			expectedSignature(second, payload.ID, timestamp, payload.Body),
		}, strings.Split(received.Header.Get("webhook-signature"), " "))
	})

	t.Run("keeps the start of the response body", func(t *testing.T) {
		t.Parallel()
		server := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(bytes.Repeat([]byte("a"), 2*maxResponseBody))
		})
		client := New(Options{
			AllowedNetworks: loopbackNetworks,
		})

		result := client.Send(t.Context(), newTestTarget(server.URL), newTestPayload(t))

		assert.Len(t, result.ResponseBody, maxResponseBody)
	})

	t.Run("classifies the response", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name           string
			status         int
			retryAfter     string
			wantOutcome    Outcome
			wantRetryAfter time.Duration
		}{
			{
				name:        "success",
				status:      http.StatusOK,
				wantOutcome: OutcomeDelivered,
			},
			{
				name:        "redirect",
				status:      http.StatusFound,
				wantOutcome: OutcomeRejected,
			},
			{
				name:        "client error",
				status:      http.StatusBadRequest,
				wantOutcome: OutcomeRejected,
			},
			{
				name:        "request timeout",
				status:      http.StatusRequestTimeout,
				wantOutcome: OutcomeRetry,
			},
			{
				name:        "gone",
				status:      http.StatusGone,
				wantOutcome: OutcomeGone,
			},
			{
				name:        "rate limited without retry-after",
				status:      http.StatusTooManyRequests,
				wantOutcome: OutcomeRetry,
			},
			{
				name:           "rate limited with retry-after",
				status:         http.StatusTooManyRequests,
				retryAfter:     "120",
				wantOutcome:    OutcomeDeferred,
				wantRetryAfter: 120 * time.Second,
			},
			{
				name:        "server error",
				status:      http.StatusInternalServerError,
				wantOutcome: OutcomeRetry,
			},
			{
				name:           "unavailable with retry-after",
				status:         http.StatusServiceUnavailable,
				retryAfter:     "30",
				wantOutcome:    OutcomeDeferred,
				wantRetryAfter: 30 * time.Second,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				server := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
					if tt.retryAfter != "" {
						w.Header().Set("Retry-After", tt.retryAfter)
					}
					// The redirect leads to a host that must not be requested
					w.Header().Set("Location", "http://127.0.0.2/")
					w.WriteHeader(tt.status)
				})
				client := New(Options{
					AllowedNetworks: loopbackNetworks,
				})

				result := client.Send(t.Context(), newTestTarget(server.URL), newTestPayload(t))

				require.NoError(t, result.Err)
				assert.Equal(t, tt.wantOutcome, result.Outcome)
				assert.Equal(t, tt.status, result.StatusCode)
				assert.Equal(t, tt.wantRetryAfter, result.RetryAfter)
			})
		}
	})

	t.Run("defers by the date in retry-after", func(t *testing.T) {
		t.Parallel()
		retryAt := time.Now().Add(2 * time.Minute)
		server := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Retry-After", retryAt.UTC().Format(http.TimeFormat))
			w.WriteHeader(http.StatusServiceUnavailable)
		})
		client := New(Options{
			AllowedNetworks: loopbackNetworks,
		})

		result := client.Send(t.Context(), newTestTarget(server.URL), newTestPayload(t))

		assert.Equal(t, OutcomeDeferred, result.Outcome)
		assert.InDelta(t, time.Until(retryAt).Seconds(), result.RetryAfter.Seconds(), 2)
	})

	t.Run("retries a target that cannot be reached", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.NotFoundHandler())
		server.Close()
		client := New(Options{
			AllowedNetworks: loopbackNetworks,
		})

		result := client.Send(t.Context(), newTestTarget(server.URL), newTestPayload(t))

		assert.Equal(t, OutcomeRetry, result.Outcome)
		assert.Zero(t, result.StatusCode)
		assert.Error(t, result.Err)
	})

	t.Run("accepts any certificate of a target that skips the verification", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		t.Cleanup(server.Close)
		client := New(Options{
			AllowedNetworks: loopbackNetworks,
		})
		target := newTestTarget(server.URL)
		target.SkipTLSVerify = true

		result := client.Send(t.Context(), target, newTestPayload(t))

		require.NoError(t, result.Err)
		assert.Equal(t, OutcomeDelivered, result.Outcome)
	})

	t.Run("retries a target with an untrusted certificate", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewTLSServer(http.NotFoundHandler())
		t.Cleanup(server.Close)
		client := New(Options{
			AllowedNetworks: loopbackNetworks,
		})

		result := client.Send(t.Context(), newTestTarget(server.URL), newTestPayload(t))

		assert.Equal(t, OutcomeRetry, result.Outcome)
		assert.ErrorIs(t, result.Err, ErrUntrustedCertificate)
	})

	t.Run("rejects an invalid target", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name    string
			target  func(serverURL string) Target
			wantErr error
		}{
			{
				name: "invalid url",
				target: func(string) Target {
					return newTestTarget("ftp://example.com/hook")
				},
				wantErr: ErrInvalidURL,
			},
			{
				name: "missing secret",
				target: func(serverURL string) Target {
					return Target{
						URL: serverURL,
					}
				},
				wantErr: ErrInvalidSecret,
			},
			{
				name: "invalid secret",
				target: func(serverURL string) Target {
					return Target{
						URL: serverURL,
						Secrets: []string{
							"secret",
						},
					}
				},
				wantErr: ErrInvalidSecret,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				var requests atomic.Int32
				server := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
					requests.Add(1)
					w.WriteHeader(http.StatusOK)
				})
				client := New(Options{
					AllowedNetworks: loopbackNetworks,
				})

				result := client.Send(t.Context(), tt.target(server.URL), newTestPayload(t))

				assert.Equal(t, OutcomeRejected, result.Outcome)
				assert.ErrorIs(t, result.Err, tt.wantErr)
				assert.Zero(t, requests.Load())
			})
		}
	})

	t.Run("rejects a blocked address", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			host string
		}{
			{
				name: "literal address",
				host: "127.0.0.1",
			},
			{
				name: "host name resolving to it",
				host: "localhost",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				var requests atomic.Int32
				server := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
					requests.Add(1)
					w.WriteHeader(http.StatusOK)
				})
				targetURL := strings.Replace(server.URL, "127.0.0.1", tt.host, 1)
				client := New(Options{})

				result := client.Send(t.Context(), newTestTarget(targetURL), newTestPayload(t))

				assert.Equal(t, OutcomeRejected, result.Outcome)
				assert.ErrorIs(t, result.Err, ErrBlockedAddress)
				assert.Zero(t, requests.Load())
			})
		}
	})
}

func TestClient_Preview(t *testing.T) {
	t.Parallel()

	t.Run("returns the signed request without sending it", func(t *testing.T) {
		t.Parallel()
		key := randomKey(generatedSecretBytes)
		payload := newTestPayload(t)
		client := New(Options{
			UserAgent: "webhook-test",
		})

		request, err := client.Preview(Target{
			URL: "https://example.com/hook",
			Secrets: []string{
				encodeSecret(key),
			},
		}, payload)

		require.NoError(t, err)
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Equal(t, "https://example.com/hook", request.URL)
		assert.Equal(t, payload.Body, request.Body)
		assert.Equal(t, "application/json", request.Header.Get("Content-Type"))
		assert.Equal(t, "webhook-test", request.Header.Get("User-Agent"))
		assert.Equal(t, payload.ID, request.Header.Get("webhook-id"))
		timestamp := request.Header.Get("webhook-timestamp")
		assert.Equal(t, expectedSignature(key, payload.ID, timestamp, payload.Body), request.Header.Get("webhook-signature"))
	})

	t.Run("rejects a target without a valid secret", func(t *testing.T) {
		t.Parallel()
		client := New(Options{})

		_, err := client.Preview(Target{
			URL: "https://example.com/hook",
		}, newTestPayload(t))

		assert.ErrorIs(t, err, ErrInvalidSecret)
	})
}

// newTestServer starts a server with the handler that closes when the test ends.
func newTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

// newTestTarget returns a target at the URL with a new secret.
func newTestTarget(targetURL string) Target {
	return Target{
		URL: targetURL,
		Secrets: []string{
			GenerateSecret(),
		},
	}
}

// newTestPayload returns the payload of a message with some data.
func newTestPayload(t *testing.T) Payload {
	t.Helper()

	payload, err := NewPayload("item.updated", map[string]int{
		"count": 4,
	})
	require.NoError(t, err)
	return payload
}

// randomKey returns a random key of the given length.
func randomKey(length int) []byte {
	key := make([]byte, length)
	_, _ = rand.Read(key)
	return key
}

// encodeSecret returns the secret of a key in the format of the specification.
func encodeSecret(key []byte) string {
	return secretPrefix + base64.StdEncoding.EncodeToString(key)
}

// expectedSignature computes a signature as the specification defines it, independent of the code under test.
func expectedSignature(key []byte, id, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(id + "." + timestamp + "." + string(body)))
	return "v1," + base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
