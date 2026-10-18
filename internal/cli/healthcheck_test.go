package cli

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/config"
)

func TestNewHealthcheckCommand(t *testing.T) {
	t.Parallel()

	t.Run("fails without an application", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.NotFoundHandler())
		port := strconv.Itoa(serverConfigFor(t, srv).Port)
		srv.Close()

		code, _, stderr := run(t, "healthcheck", "--server-host", "127.0.0.1", "--server-port", port)

		assert.Equal(t, 1, code)
		assert.NotEmpty(t, stderr)
	})
}

func TestCheckHealth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		status  int
		useTLS  bool
		wantErr bool
	}{
		{
			name:   "accepts a healthy application",
			status: http.StatusOK,
		},
		{
			name:   "accepts a self-signed certificate",
			status: http.StatusOK,
			useTLS: true,
		},
		{
			name:    "rejects an unhealthy application",
			status:  http.StatusServiceUnavailable,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/healthz", r.URL.Path)
				w.WriteHeader(tt.status)
			})
			srv := httptest.NewServer(handler)
			if tt.useTLS {
				srv.Close()
				srv = httptest.NewTLSServer(handler)
			}
			t.Cleanup(srv.Close)

			err := checkHealth(t.Context(), serverConfigFor(t, srv))

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}

	t.Run("rejects a missing application", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.NotFoundHandler())
		cfg := serverConfigFor(t, srv)
		srv.Close()

		err := checkHealth(t.Context(), cfg)

		assert.Error(t, err)
	})
}

// serverConfigFor returns the server settings that point to a test server in place of the application.
func serverConfigFor(t *testing.T, srv *httptest.Server) config.ServerConfig {
	t.Helper()

	host, port, err := net.SplitHostPort(srv.Listener.Addr().String())
	require.NoError(t, err)
	portNumber, err := strconv.Atoi(port)
	require.NoError(t, err)
	return config.ServerConfig{
		Host:       host,
		Port:       portNumber,
		SSLEnabled: srv.TLS != nil,
	}
}
