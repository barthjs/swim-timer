package serial

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
)

const (
	// retryInterval is the pause between two attempts to find or open a port.
	retryInterval = 5 * time.Second
)

var (
	// ErrNotConnected is returned by Ping when no port is open.
	ErrNotConnected = errors.New("serial: not connected")
)

// Handler consumes an opened connection until reading from it fails or ctx is canceled.
// The returned error describes why the connection ended.
type Handler func(ctx context.Context, r io.Reader) error

// Supervisor keeps a connection to a serial port open and reconnects after it drops.
// Without a configured port, it scans the system for ports and tries all of them in order,
// using the first one that opens.
type Supervisor struct {
	logger zerolog.Logger
	port   string
	mode   Mode

	listPorts     func() ([]string, error)
	retryInterval time.Duration

	// started guards Run, since two loops would compete for the same port.
	started atomic.Bool

	mu        sync.Mutex
	connected string
}

// Option configures a Supervisor.
type Option func(*Supervisor)

// WithPortLister replaces the scan of the system's serial ports, unless list is nil.
func WithPortLister(list func() ([]string, error)) Option {
	return func(s *Supervisor) {
		if list != nil {
			s.listPorts = list
		}
	}
}

// WithRetryInterval sets the pause between two attempts to find or open a port.
// A duration that is not positive is ignored.
func WithRetryInterval(d time.Duration) Option {
	return func(s *Supervisor) {
		if d > 0 {
			s.retryInterval = d
		}
	}
}

// New creates a Supervisor for the given port, or for every port found on the system if the port is empty.
func New(logger zerolog.Logger, port string, mode Mode, opts ...Option) *Supervisor {
	s := &Supervisor{
		logger:        logger,
		port:          port,
		mode:          mode,
		listPorts:     listPorts,
		retryInterval: retryInterval,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Run opens a port, hands it to handle, and reopens it after handle returns, until ctx is canceled.
// It may only be called once per Supervisor, a second call logs an error and returns.
func (s *Supervisor) Run(ctx context.Context, handle Handler) {
	if !s.started.CompareAndSwap(false, true) {
		s.logger.Error().Msg("serial supervisor is already running")
		return
	}

	ports, ok := s.candidates(ctx)
	if !ok {
		return
	}

	for {
		if !s.connectAndHandle(ctx, ports, handle) && s.port == "" {
			// None of the known ports could be opened, for example because a USB adapter was plugged into another socket
			// and got a new name, so the system is scanned again.
			ports, ok = s.candidates(ctx)
			if !ok {
				return
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(s.retryInterval):
		}
	}
}

// Ping reports whether a port is currently open.
func (s *Supervisor) Ping(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.connected == "" {
		return ErrNotConnected
	}
	return nil
}

// Port returns the name of the currently open port, or an empty string while no port is open.
func (s *Supervisor) Port() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.connected
}

// candidates returns the configured port or scans the system until it finds one, and false once ctx is canceled.
func (s *Supervisor) candidates(ctx context.Context) ([]string, bool) {
	if s.port != "" {
		return []string{s.port}, true
	}

	for {
		found, err := s.listPorts()
		if err != nil {
			s.logger.Error().Err(err).Msg("failed to list serial ports")
		} else if len(found) == 0 {
			s.logger.Warn().Dur("retry_interval", s.retryInterval).Msg("no serial ports found")
		} else {
			return found, true
		}

		select {
		case <-ctx.Done():
			return nil, false
		case <-time.After(s.retryInterval):
		}
	}
}

// connectAndHandle hands the first port that opens to handle and reports whether one opened.
func (s *Supervisor) connectAndHandle(ctx context.Context, ports []string, handle Handler) bool {
	var conn io.ReadCloser
	var name string

	for _, candidate := range ports {
		if ctx.Err() != nil {
			return false
		}

		opened, err := open(ctx, candidate, s.mode)
		if err != nil {
			s.logger.Debug().Err(err).Str("port", candidate).Msg("port busy or cannot open")
			continue
		}

		conn = opened
		name = candidate
		break
	}

	if conn == nil {
		s.logger.Warn().Dur("retry_interval", s.retryInterval).Msg("no available serial port")
		return false
	}

	s.logger.Info().Str("port", name).Msg("connected to serial port")
	s.setConnected(name)
	defer s.setConnected("")

	// Closing the port on cancel keeps a read that blocks on a silent device from outliving a shutdown
	done := make(chan struct{})
	defer close(done)

	go func() {
		select {
		case <-ctx.Done():
		case <-done:
		}
		_ = conn.Close()
	}()

	if err := handle(ctx, conn); err != nil && ctx.Err() == nil {
		s.logger.Error().Err(err).Str("port", name).Msg("error reading from serial port")
	}
	return true
}

// setConnected records the name of the currently open port, or empty if none is open.
func (s *Supervisor) setConnected(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.connected = name
}
