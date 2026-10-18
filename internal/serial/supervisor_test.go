package serial

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scriptedLister answers the port scans with the given results in order and repeats the last one.
type scriptedLister struct {
	mu      sync.Mutex
	results []scanResult
	calls   int
}

// scanResult is the outcome of one scripted port scan.
type scanResult struct {
	ports []string
	err   error
}

func TestWithPortLister(t *testing.T) {
	t.Parallel()

	t.Run("ignores a nil lister", func(t *testing.T) {
		t.Parallel()

		supervisor := New(zerolog.Nop(), "", Mode{}, WithPortLister(nil))

		assert.NotNil(t, supervisor.listPorts)
	})
}

func TestWithRetryInterval(t *testing.T) {
	t.Parallel()

	t.Run("ignores a non-positive interval", func(t *testing.T) {
		t.Parallel()

		supervisor := New(zerolog.Nop(), "", Mode{}, WithRetryInterval(0))

		assert.Equal(t, retryInterval, supervisor.retryInterval)
	})
}

func TestSupervisor_Run(t *testing.T) {
	t.Parallel()

	t.Run("connects and reports the port", func(t *testing.T) {
		t.Parallel()

		address := listen(t, []byte("data"))
		supervisor := New(zerolog.Nop(), address, Mode{})
		assert.ErrorIs(t, supervisor.Ping(t.Context()), ErrNotConnected, "before Run")
		ctx, cancel := context.WithCancel(t.Context())
		received := make(chan string, 1)
		portDuringHandle := make(chan string, 1)
		stopped := make(chan struct{})

		go func() {
			supervisor.Run(ctx, func(ctx context.Context, r io.Reader) error {
				portDuringHandle <- supervisor.Port()
				buf := make([]byte, 4)
				if _, err := io.ReadFull(r, buf); err != nil {
					return err
				}
				received <- string(buf)
				// Block like a real handler until the supervisor closes the connection on shutdown
				_, err := io.ReadAll(r)
				return err
			})
			close(stopped)
		}()

		select {
		case port := <-portDuringHandle:
			assert.Equal(t, address, port)
		case <-time.After(2 * time.Second):
			require.FailNow(t, "handler was not called")
		}
		select {
		case data := <-received:
			assert.Equal(t, "data", data)
		case <-time.After(2 * time.Second):
			require.FailNow(t, "handler received no data")
		}
		assert.NoError(t, supervisor.Ping(t.Context()), "while connected")
		cancel()
		select {
		case <-stopped:
		case <-time.After(2 * time.Second):
			require.FailNow(t, "Run did not return after cancel")
		}
		assert.Empty(t, supervisor.Port(), "after Run")
	})

	t.Run("connects to the first scanned port that opens", func(t *testing.T) {
		t.Parallel()

		live := listen(t, []byte("data"))
		lister := &scriptedLister{
			results: []scanResult{
				{
					ports: []string{"127.0.0.1:1", live},
				},
			},
		}
		supervisor := New(zerolog.Nop(), "", Mode{}, WithPortLister(lister.list), WithRetryInterval(time.Millisecond))

		port := runUntilConnected(t, supervisor)

		assert.Equal(t, live, port, "the first port that opens")
	})

	t.Run("retries the scan", func(t *testing.T) {
		t.Parallel()

		live := listen(t, []byte("data"))
		lister := &scriptedLister{
			results: []scanResult{
				{
					err: errors.New("enumeration failed"),
				},
				{
					ports: nil,
				},
				{
					ports: []string{live},
				},
			},
		}
		supervisor := New(zerolog.Nop(), "", Mode{}, WithPortLister(lister.list), WithRetryInterval(time.Millisecond))

		port := runUntilConnected(t, supervisor)

		assert.Equal(t, live, port)
		assert.Equal(t, 3, lister.count(), "after a failed and an empty scan")
	})

	t.Run("scans again when no known port opens", func(t *testing.T) {
		t.Parallel()

		live := listen(t, []byte("data"))
		lister := &scriptedLister{
			results: []scanResult{
				{
					ports: []string{"127.0.0.1:1"},
				},
				{
					ports: []string{live},
				},
			},
		}
		supervisor := New(zerolog.Nop(), "", Mode{}, WithPortLister(lister.list), WithRetryInterval(time.Millisecond))

		port := runUntilConnected(t, supervisor)

		assert.Equal(t, live, port, "for example after a USB adapter got a new name")
	})

	t.Run("skips the scan for a configured port", func(t *testing.T) {
		t.Parallel()

		live := listen(t, []byte("data"))
		lister := &scriptedLister{
			results: []scanResult{
				{
					ports: []string{"127.0.0.1:1"},
				},
			},
		}
		supervisor := New(zerolog.Nop(), live, Mode{}, WithPortLister(lister.list))

		port := runUntilConnected(t, supervisor)

		assert.Equal(t, live, port)
		assert.Zero(t, lister.count())
	})

	t.Run("runs once", func(t *testing.T) {
		t.Parallel()

		supervisor := New(zerolog.Nop(), "127.0.0.1:1", Mode{})
		supervisor.started.Store(true)
		done := make(chan struct{})

		go func() {
			supervisor.Run(t.Context(), func(context.Context, io.Reader) error { return nil })
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(time.Second):
			require.FailNow(t, "second Run did not return immediately")
		}
	})

	t.Run("stops without an open port", func(t *testing.T) {
		t.Parallel()

		// Port 1 on the loopback interface refuses connections, so no port can be opened
		supervisor := New(zerolog.Nop(), "127.0.0.1:1", Mode{})
		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		defer cancel()
		done := make(chan struct{})

		go func() {
			supervisor.Run(ctx, func(context.Context, io.Reader) error {
				assert.Fail(t, "handler called without an open port")
				return nil
			})
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(2 * time.Second):
			require.FailNow(t, "Run did not return after the context ended")
		}
		assert.ErrorIs(t, supervisor.Ping(t.Context()), ErrNotConnected)
	})

	t.Run("stops during the scan", func(t *testing.T) {
		t.Parallel()

		lister := &scriptedLister{
			results: []scanResult{
				{
					ports: nil,
				},
			},
		}
		supervisor := New(zerolog.Nop(), "", Mode{}, WithPortLister(lister.list), WithRetryInterval(time.Millisecond))
		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()
		done := make(chan struct{})

		go func() {
			supervisor.Run(ctx, func(context.Context, io.Reader) error { return nil })
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "Run did not return while no port was found")
		}
	})
}

// list returns the next scripted scan result and repeats the last one when the script is exhausted.
func (l *scriptedLister) list() ([]string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	result := l.results[min(l.calls, len(l.results)-1)]
	l.calls++
	return result.ports, result.err
}

// runUntilConnected runs the supervisor until its handler reads from a port and returns that port.
func runUntilConnected(t *testing.T, supervisor *Supervisor) string {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	connected := make(chan string, 1)
	stopped := make(chan struct{})

	go func() {
		supervisor.Run(ctx, func(ctx context.Context, r io.Reader) error {
			select {
			case connected <- supervisor.Port():
			default:
			}
			<-ctx.Done()
			return ctx.Err()
		})
		close(stopped)
	}()

	var port string
	select {
	case port = <-connected:
	case <-time.After(5 * time.Second):
		cancel()
		require.FailNow(t, "the supervisor did not connect")
	}

	cancel()
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "Run did not return after cancel")
	}
	return port
}

// count returns the number of scans.
func (l *scriptedLister) count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.calls
}
