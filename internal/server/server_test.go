package server_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/server"
)

func TestServer_Run(t *testing.T) {
	t.Parallel()

	t.Run("serves until canceled", func(t *testing.T) {
		t.Parallel()

		port := freePort(t)
		srv := server.New(zerolog.Nop(), server.Options{
			Host: "127.0.0.1",
			Port: port,
		}, helloHandler())
		ctx, cancel := context.WithCancel(t.Context())
		result := make(chan error, 1)

		go func() { result <- srv.Run(ctx) }()

		response := waitForServer(t, http.DefaultClient, srv.URL())
		body, err := io.ReadAll(response.Body)
		require.NoError(t, err)
		_ = response.Body.Close()
		assert.Equal(t, "hello", string(body))
		cancel()
		select {
		case err := <-result:
			assert.NoError(t, err)
		case <-time.After(5 * time.Second):
			require.FailNow(t, "Run did not return after cancel")
		}
	})

	t.Run("ends streams on shutdown", func(t *testing.T) {
		t.Parallel()

		port := freePort(t)
		streaming := make(chan struct{})
		srv := server.New(zerolog.Nop(), server.Options{
			Host: "127.0.0.1",
			Port: port,
		}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/stream" {
				_, _ = io.WriteString(w, "hello")
				return
			}
			_ = http.NewResponseController(w).Flush()
			close(streaming)
			// A stream only ends with its request context
			<-r.Context().Done()
		}))
		ctx, cancel := context.WithCancel(t.Context())
		result := make(chan error, 1)
		go func() { result <- srv.Run(ctx) }()
		ready := waitForServer(t, http.DefaultClient, srv.URL())
		_ = ready.Body.Close()
		go func() {
			request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL()+"/stream", nil)
			if err != nil {
				return
			}
			response, err := http.DefaultClient.Do(request)
			if err == nil {
				_, _ = io.Copy(io.Discard, response.Body)
				_ = response.Body.Close()
			}
		}()
		<-streaming
		started := time.Now()

		cancel()

		select {
		case err := <-result:
			assert.NoError(t, err)
			assert.Less(t, time.Since(started), 15*time.Second, "the stream does not hold the shutdown until its timeout")
		case <-time.After(20 * time.Second):
			require.FailNow(t, "Run did not return while a stream was open")
		}
	})

	t.Run("rejects a busy port", func(t *testing.T) {
		t.Parallel()

		var config net.ListenConfig
		listener, err := config.Listen(t.Context(), "tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer func() { _ = listener.Close() }()
		address, ok := listener.Addr().(*net.TCPAddr)
		require.True(t, ok)
		srv := server.New(zerolog.Nop(), server.Options{
			Host: "127.0.0.1",
			Port: address.Port,
		}, helloHandler())

		err = srv.Run(t.Context())

		assert.Error(t, err)
	})
}

// freePort returns a TCP port that was free a moment ago.
func freePort(t *testing.T) int {
	t.Helper()

	var config net.ListenConfig
	listener, err := config.Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address, ok := listener.Addr().(*net.TCPAddr)
	require.True(t, ok)
	require.NoError(t, listener.Close())
	return address.Port
}

// waitForServer polls the URL until the server answers.
func waitForServer(t *testing.T, client *http.Client, url string) *http.Response {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
		require.NoError(t, err)
		response, err := client.Do(request)
		if err == nil {
			return response
		}
		time.Sleep(10 * time.Millisecond)
	}
	require.FailNow(t, "server did not start", url)
	return nil
}

// helloHandler returns a handler that answers with hello.
func helloHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "hello")
	})
}

func TestServer_URL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		options server.Options
		want    string
	}{
		{
			name: "returns localhost for all interfaces",
			options: server.Options{
				Host: "",
				Port: 8080,
			},
			want: "http://localhost:8080",
		},
		{
			name: "returns localhost for the wildcard",
			options: server.Options{
				Host: "*",
				Port: 8080,
			},
			want: "http://localhost:8080",
		},
		{
			name: "returns localhost for the IPv4 any address",
			options: server.Options{
				Host: "0.0.0.0",
				Port: 80,
			},
			want: "http://localhost:80",
		},
		{
			name: "returns localhost for the IPv6 any address",
			options: server.Options{
				Host: "::",
				Port: 80,
			},
			want: "http://localhost:80",
		},
		{
			name: "returns a specific IPv4 address",
			options: server.Options{
				Host: "192.168.1.10",
				Port: 8080,
			},
			want: "http://192.168.1.10:8080",
		},
		{
			name: "puts an IPv6 address in brackets",
			options: server.Options{
				Host: "::1",
				Port: 8080,
			},
			want: "http://[::1]:8080",
		},
		{
			name: "returns the hostname",
			options: server.Options{
				Host: "example.test",
				Port: 9000,
			},
			want: "http://example.test:9000",
		},
		{
			name: "returns https with TLS",
			options: server.Options{
				Host:       "",
				Port:       8443,
				SSLEnabled: true,
			},
			want: "https://localhost:8443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := server.New(zerolog.Nop(), tt.options, nil)

			url := srv.URL()

			assert.Equal(t, tt.want, url)
		})
	}
}
