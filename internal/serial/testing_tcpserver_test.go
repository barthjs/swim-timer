package serial

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

// listen starts a TCP server on a free local port that sends payload to every client
// and keeps the connection open until the test ends.
func listen(t *testing.T, payload []byte) string {
	t.Helper()

	var config net.ListenConfig
	listener, err := config.Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			_, _ = conn.Write(payload)
			t.Cleanup(func() { _ = conn.Close() })
		}
	}()

	return listener.Addr().String()
}
