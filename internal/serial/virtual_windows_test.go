package serial

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenVirtual(t *testing.T) {
	t.Parallel()

	t.Run("relays writes to a connected reader", func(t *testing.T) {
		t.Parallel()

		writer, address, err := OpenVirtual()
		require.NoError(t, err)
		t.Cleanup(func() { _ = writer.Close() })

		reader := dialRelay(t, writer, address)

		_, err = writer.Write([]byte("frame"))
		require.NoError(t, err)

		buf := make([]byte, 5)
		_, err = io.ReadFull(reader, buf)
		require.NoError(t, err)
		assert.Equal(t, "frame", string(buf))
	})
}

func TestRelayWriter_Close(t *testing.T) {
	t.Parallel()

	t.Run("releases the relay", func(t *testing.T) {
		t.Parallel()

		writer, address, err := OpenVirtual()
		require.NoError(t, err)
		reader := dialRelay(t, writer, address)

		require.NoError(t, writer.Close())

		require.NoError(t, reader.SetReadDeadline(time.Now().Add(5*time.Second)))
		_, err = reader.Read(make([]byte, 1))
		assert.ErrorIs(t, err, io.EOF, "the relayed connection is closed")

		_, err = net.DialTimeout("tcp", address, time.Second)
		assert.Error(t, err, "the listener is closed")
		assert.NoError(t, writer.Close(), "a second close does nothing")
	})

	t.Run("ends a write to a reader that stopped reading", func(t *testing.T) {
		t.Parallel()

		writer, address, err := OpenVirtual()
		require.NoError(t, err)
		dialRelay(t, writer, address)
		relay, ok := writer.(*relayWriter)
		require.True(t, ok)

		// The reader never reads, so the relay and then the writer block once the socket buffers are full
		written := make(chan error, 1)
		go func() {
			_, err := writer.Write(make([]byte, 64<<20))
			written <- err
		}()
		require.Eventually(t, func() bool {
			if relay.mu.TryLock() {
				relay.mu.Unlock()
				return false
			}
			return true
		}, 5*time.Second, time.Millisecond, "the write is pending")

		closed := make(chan error, 1)
		go func() { closed <- writer.Close() }()

		select {
		case err := <-closed:
			assert.NoError(t, err)
		case <-time.After(5 * time.Second):
			require.Fail(t, "close waits for the pending write")
		}
		select {
		case err := <-written:
			assert.Error(t, err, "the pending write ends with the closed connection")
		case <-time.After(5 * time.Second):
			require.Fail(t, "the pending write never ends")
		}
	})
}

// dialRelay connects a reader to the relay of writer and waits until the relay has registered it,
// which happens after Accept and so may come after Dial returns.
func dialRelay(t *testing.T, writer io.WriteCloser, address string) net.Conn {
	t.Helper()

	relay, ok := writer.(*relayWriter)
	require.True(t, ok)

	conn, err := net.Dial("tcp", address)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	// The writer itself is the first connection of the relay
	require.Eventually(t, func() bool {
		relay.relay.mu.Lock()
		defer relay.relay.mu.Unlock()
		return len(relay.relay.conns) == 2
	}, 5*time.Second, 10*time.Millisecond)
	return conn
}
