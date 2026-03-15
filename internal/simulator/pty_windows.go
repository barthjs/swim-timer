//go:build windows

package simulator

import (
	"fmt"
	"io"
	"net"
	"sync"
)

type tcpRelay struct {
	listeners map[net.Conn]bool
	mu        *sync.Mutex
}

// OpenPTY creates a simulated serial connection on Windows using a local TCP listener.
// It returns a writer (the connection) and the address that can be used as a serial port substitute.
// The listener accepts multiple connections and relays data between them.
func OpenPTY() (io.WriteCloser, string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, "", fmt.Errorf("failed to create listener: %w", err)
	}

	addr := listener.Addr().String()

	// Use a pipe-like structure to relay data between simulator and connector
	relay := &tcpRelay{
		listeners: make(map[net.Conn]bool),
		mu:        &sync.Mutex{},
	}

	go func() {
		defer func(listener net.Listener) {
			_ = listener.Close()
		}(listener)
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}

			relay.mu.Lock()
			relay.listeners[conn] = true
			relay.mu.Unlock()

			// Read from this connection and relay to all others
			go func(c net.Conn) {
				defer func() {
					relay.mu.Lock()
					delete(relay.listeners, c)
					relay.mu.Unlock()
					_ = c.Close()
				}()

				// Copy data from this connection to all other listeners
				buf := make([]byte, 1024)
				for {
					n, err := c.Read(buf)
					if err != nil {
						return
					}

					relay.mu.Lock()
					// Track listeners to remove if write fails
					var failedListeners []net.Conn
					for listener := range relay.listeners {
						if listener != c {
							_, err := listener.Write(buf[:n])
							if err != nil {
								failedListeners = append(failedListeners, listener)
							}
						}
					}
					// Remove listeners that failed to write
					for _, failedListener := range failedListeners {
						delete(relay.listeners, failedListener)
						_ = failedListener.Close()
					}
					relay.mu.Unlock()
				}
			}(conn)
		}
	}()

	// Create a connection that the simulator will write to
	simConn, err := net.Dial("tcp", addr)
	if err != nil {
		_ = listener.Close()
		return nil, "", fmt.Errorf("failed to connect simulator: %w", err)
	}

	// Set TCP options for better Windows compatibility
	tcpConn, ok := simConn.(*net.TCPConn)
	if !ok {
		_ = simConn.Close()
		_ = listener.Close()
		return nil, "", fmt.Errorf("unexpected connection type")
	}

	if err := tcpConn.SetNoDelay(true); err != nil {
		_ = simConn.Close()
		_ = listener.Close()
		return nil, "", fmt.Errorf("failed to set no delay: %w", err)
	}

	if err := tcpConn.SetKeepAlive(true); err != nil {
		_ = simConn.Close()
		_ = listener.Close()
		return nil, "", fmt.Errorf("failed to set keep alive: %w", err)
	}

	wrapped := &wrappedConn{conn: simConn}
	return wrapped, addr, nil
}

// wrappedConn wraps a net.Conn to provide io.WriteCloser interface
type wrappedConn struct {
	conn net.Conn
	mu   sync.Mutex
}

func (wc *wrappedConn) Write(p []byte) (n int, err error) {
	wc.mu.Lock()
	defer wc.mu.Unlock()
	if wc.conn == nil {
		return 0, io.ErrClosedPipe
	}
	return wc.conn.Write(p)
}

func (wc *wrappedConn) Close() error {
	wc.mu.Lock()
	defer wc.mu.Unlock()
	if wc.conn != nil {
		err := wc.conn.Close()
		wc.conn = nil
		return err
	}
	return nil
}
