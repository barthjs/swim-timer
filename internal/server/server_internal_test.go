package server

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_Run(t *testing.T) {
	t.Parallel()

	t.Run("serves TLS", func(t *testing.T) {
		t.Parallel()

		certFile, keyFile := newCert(t, "localhost", "127.0.0.1")
		srv := New(
			zerolog.Nop(),
			Options{
				Host:        "127.0.0.1",
				Port:        freeTCPPort(t),
				SSLEnabled:  true,
				SSLCertFile: certFile,
				SSLKeyFile:  keyFile,
			},
			http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "secure") }),
		)
		ctx, cancel := context.WithCancel(t.Context())
		result := make(chan error, 1)
		client := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}

		go func() { result <- srv.Run(ctx) }()

		var body []byte
		var state *tls.ConnectionState
		require.Eventually(t, func() bool {
			request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL(), nil)
			if err != nil {
				return false
			}
			response, err := client.Do(request)
			if err != nil {
				return false
			}
			defer func() { _ = response.Body.Close() }()
			state = response.TLS
			body, err = io.ReadAll(response.Body)
			return err == nil
		}, 5*time.Second, 10*time.Millisecond)
		assert.Equal(t, "secure", string(body))
		assert.NotNil(t, state)
		cancel()
		assert.NoError(t, <-result)
	})

	t.Run("rejects an unusable certificate", func(t *testing.T) {
		t.Parallel()

		missing := filepath.Join(t.TempDir(), "missing.pem")
		srv := New(
			zerolog.Nop(),
			Options{
				Host:        "127.0.0.1",
				Port:        freeTCPPort(t),
				SSLEnabled:  true,
				SSLCertFile: missing,
				SSLKeyFile:  missing,
			}, http.NotFoundHandler(),
		)

		err := srv.Run(t.Context())

		assert.Error(t, err)
	})
}

// freeTCPPort returns a TCP port that was free a moment ago.
func freeTCPPort(t *testing.T) int {
	t.Helper()

	var config net.ListenConfig
	listener, err := config.Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address, ok := listener.Addr().(*net.TCPAddr)
	require.True(t, ok)
	require.NoError(t, listener.Close())
	return address.Port
}

func TestServer_logStartup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		host     string
		wantAddr string
	}{
		{
			name:     "logs the wildcard address for all interfaces",
			host:     "",
			wantAddr: `"addr":"*:8080"`,
		},
		{
			name:     "logs the address of a specific host",
			host:     "127.0.0.1",
			wantAddr: `"addr":"127.0.0.1:8080"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var output bytes.Buffer
			srv := New(zerolog.New(&output), Options{
				Host: tt.host,
				Port: 8080,
			}, nil)

			srv.logStartup()

			assert.Contains(t, output.String(), tt.wantAddr)
			assert.Contains(t, output.String(), srv.URL())
		})
	}
}
