package serial

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
)

// tcpRelay forwards everything read from one of its connections to all the others.
type tcpRelay struct {
	listener net.Listener

	mu     sync.Mutex
	conns  map[net.Conn]bool
	closed bool
}

// OpenVirtual creates a virtual serial port backed by a local TCP relay, since Windows has no pseudo-terminals.
// Bytes written to the returned writer can be read by opening the returned host:port name,
// which selects the TCP driver (see driverFor).
func OpenVirtual() (io.WriteCloser, string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, "", fmt.Errorf("serial: creating listener: %w", err)
	}

	addr := listener.Addr().String()

	relay := &tcpRelay{
		listener: listener,
		conns:    make(map[net.Conn]bool),
	}
	go relay.serve()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		_ = relay.close()
		return nil, "", fmt.Errorf("serial: connecting to relay: %w", err)
	}

	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		_ = conn.Close()
		_ = relay.close()
		return nil, "", fmt.Errorf("serial: unexpected connection type")
	}

	if err := tcpConn.SetNoDelay(true); err != nil {
		_ = conn.Close()
		_ = relay.close()
		return nil, "", fmt.Errorf("serial: setting no delay: %w", err)
	}

	if err := tcpConn.SetKeepAlive(true); err != nil {
		_ = conn.Close()
		_ = relay.close()
		return nil, "", fmt.Errorf("serial: setting keep alive: %w", err)
	}

	return newRelayWriter(conn, relay), addr, nil
}

// serve accepts connections until the listener fails or the relay is closed and relays each of them.
func (r *tcpRelay) serve() {
	defer func() { _ = r.listener.Close() }()

	for {
		conn, err := r.listener.Accept()
		if err != nil {
			return
		}

		r.mu.Lock()
		if r.closed {
			r.mu.Unlock()
			_ = conn.Close()
			return
		}
		r.conns[conn] = true
		r.mu.Unlock()

		go r.relay(conn)
	}
}

// relay copies everything read from conn to all other connections,
// dropping the connections that fail to accept a write.
func (r *tcpRelay) relay(conn net.Conn) {
	defer func() {
		r.mu.Lock()
		delete(r.conns, conn)
		r.mu.Unlock()
		_ = conn.Close()
	}()

	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			return
		}

		// The writes run without the lock, since a reader that stops reading blocks them until close releases it
		var failed []net.Conn
		for _, other := range r.others(conn) {
			if _, err := other.Write(buf[:n]); err != nil {
				failed = append(failed, other)
			}
		}

		r.mu.Lock()
		for _, other := range failed {
			delete(r.conns, other)
		}
		r.mu.Unlock()
		for _, other := range failed {
			_ = other.Close()
		}
	}
}

// others returns the relayed connections except conn.
func (r *tcpRelay) others(conn net.Conn) []net.Conn {
	r.mu.Lock()
	defer r.mu.Unlock()

	others := make([]net.Conn, 0, len(r.conns))
	for other := range r.conns {
		if other != conn {
			others = append(others, other)
		}
	}
	return others
}

// close closes the listener and every relayed connection, so the port and the readers are released.
// Closing a connection also ends a write to it that blocks, since its reader stopped reading.
func (r *tcpRelay) close() error {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil
	}
	r.closed = true
	conns := make([]net.Conn, 0, len(r.conns))
	for conn := range r.conns {
		conns = append(conns, conn)
		delete(r.conns, conn)
	}
	r.mu.Unlock()

	err := r.listener.Close()
	for _, conn := range conns {
		_ = conn.Close()
	}
	return err
}

// relayWriter serializes writes to the relay connection and makes Close idempotent.
// Close does not wait for a pending write, it closes the connection under it, which ends the write.
type relayWriter struct {
	mu     sync.Mutex
	conn   net.Conn
	relay  *tcpRelay
	closed atomic.Bool
	close  func() error
}

// newRelayWriter creates the writer of the relay that sends through conn.
func newRelayWriter(conn net.Conn, relay *tcpRelay) *relayWriter {
	w := &relayWriter{
		conn:  conn,
		relay: relay,
	}
	w.close = sync.OnceValue(func() error {
		w.closed.Store(true)
		return errors.Join(w.conn.Close(), w.relay.close())
	})
	return w
}

// Write sends p to the client connected to the relay.
func (w *relayWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed.Load() {
		return 0, io.ErrClosedPipe
	}
	return w.conn.Write(p)
}

// Close stops the relay and closes its connections, a later call returns the result of the first.
func (w *relayWriter) Close() error {
	return w.close()
}
