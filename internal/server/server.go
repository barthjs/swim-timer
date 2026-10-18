package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/rs/zerolog"

	"github.com/barthjs/swim-timer/internal/appdirs"
)

// Defaults for every request. Routes that need more raise them with BodyLimit and Timeout.
const (
	DefaultMaxBodySize    = 1 << 20
	DefaultRequestTimeout = 30 * time.Second
)

const (
	// shutdownTimeout bounds how long a shutdown waits for the running requests.
	shutdownTimeout = 30 * time.Second
	// shutdownGrace is how long a shutdown lets the running requests finish on their own before it cancels their contexts.
	// Without the cancel, a stream that only ends with its client would hold the shutdown for the whole shutdownTimeout.
	shutdownGrace = 5 * time.Second
)

// Options holds the settings of a Server.
type Options struct {
	Host        string
	Port        int
	SSLEnabled  bool
	SSLCertFile string
	SSLKeyFile  string
	AppName     string
	// CertDirs are the config roots a generated certificate is searched for and stored in,
	// below AppName. The zero value uses the roots of the current OS (see appdirs.ConfigRoots).
	CertDirs appdirs.Roots
}

// Server serves an HTTP handler and shuts it down gracefully.
type Server struct {
	options Options
	logger  zerolog.Logger
	handler http.Handler
}

// New creates a Server.
func New(logger zerolog.Logger, options Options, handler http.Handler) *Server {
	return &Server{
		options: options,
		logger:  logger,
		handler: handler,
	}
}

// Run starts the server and blocks until ctx ends and the server has shut down.
func (s *Server) Run(ctx context.Context) error {
	// The requests run on their own base context, so a shutdown can end them after shutdownGrace
	requestCtx, cancelRequests := context.WithCancel(context.Background())
	defer cancelRequests()

	srv := &http.Server{
		Addr:              net.JoinHostPort(s.options.Host, strconv.Itoa(s.options.Port)),
		Handler:           s.handler,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      DefaultRequestTimeout,
		IdleTimeout:       time.Minute,
		BaseContext: func(net.Listener) context.Context {
			return requestCtx
		},
	}

	shutdownDone := make(chan struct{})
	// failed ends the shutdown goroutine when the server could not start
	failed := make(chan struct{})

	go func() {
		defer close(shutdownDone)

		select {
		case <-failed:
			return
		case <-ctx.Done():
			s.logger.Info().Msg("shutting down server")
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		grace := time.AfterFunc(shutdownGrace, cancelRequests)
		defer grace.Stop()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			s.logger.Error().Err(err).Msg("error during server shutdown")
		}

		s.logger.Info().Str("addr", srv.Addr).Msg("shutdown complete")
	}()

	s.logStartup()

	var err error
	if s.options.SSLEnabled {
		var certFile, keyFile string
		certFile, keyFile, err = s.certFiles()
		if err != nil {
			close(failed)
			<-shutdownDone
			return fmt.Errorf("resolving TLS certificate: %w", err)
		}
		err = srv.ListenAndServeTLS(certFile, keyFile)
	} else {
		err = srv.ListenAndServe()
	}
	if !errors.Is(err, http.ErrServerClosed) {
		close(failed)
		<-shutdownDone
		return err
	}

	// Wait for the shutdown goroutine to finish
	<-shutdownDone

	s.logger.Info().Str("addr", srv.Addr).Msg("stopped server")

	return nil
}

// URL returns a clickable server URL for the user display. If the server binds to all interfaces, localhost is used.
func (s *Server) URL() string {
	scheme := "http"
	if s.options.SSLEnabled {
		scheme = "https"
	}

	displayHost := s.options.Host
	if allInterfaces(displayHost) {
		displayHost = "localhost"
	}

	if ip := net.ParseIP(displayHost); ip != nil && ip.To4() == nil {
		// IPv6
		return fmt.Sprintf("%s://[%s]:%d", scheme, displayHost, s.options.Port)
	}
	// IPv4 or localhost
	return fmt.Sprintf("%s://%s:%d", scheme, displayHost, s.options.Port)
}

// logStartup prints the startup log for the server. It shows `*:port` if the host is empty (all interfaces).
func (s *Server) logStartup() {
	addrLog := net.JoinHostPort(s.options.Host, strconv.Itoa(s.options.Port))
	if s.options.Host == "" {
		addrLog = fmt.Sprintf("*:%d", s.options.Port)
	}
	s.logger.Info().Str("addr", addrLog).Msg("starting server")
	s.logger.Info().Msgf("server accessible at %s", s.URL())
}

// allInterfaces reports whether host stands for every network interface.
func allInterfaces(host string) bool {
	switch host {
	case "", "*", "0.0.0.0", "::":
		return true
	default:
		return false
	}
}
