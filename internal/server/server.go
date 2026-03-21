package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog"

	"github.com/barthjs/swim-timer/internal/config"
)

type Server struct {
	host    string
	port    int
	logger  zerolog.Logger
	handler http.Handler
	wg      *sync.WaitGroup
}

func New(
	cfg config.Server,
	logger zerolog.Logger,
	handler http.Handler,
	wg *sync.WaitGroup,
) *Server {
	return &Server{
		host:    cfg.Host,
		port:    cfg.Port,
		logger:  logger,
		handler: handler,
		wg:      wg,
	}
}

func (s *Server) Run(ctx context.Context) error {
	srv := &http.Server{
		Addr:         net.JoinHostPort(s.host, strconv.Itoa(s.port)),
		Handler:      s.handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 20 * time.Second,
		IdleTimeout:  time.Minute,
	}

	shutdownDone := make(chan struct{})

	go func() {
		defer close(shutdownDone)

		// Listen for OS signals for graceful shutdown
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sigCh)

		select {
		case <-ctx.Done():
			s.logger.Info().Msg("context cancelled: shutting down server")
		case si := <-sigCh:
			s.logger.Info().Str("signal", si.String()).Msg("caught signal: shutting down server")
		}

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			s.logger.Error().Err(err).Msg("error during server shutdown")
		}

		s.logger.Info().Str("addr", srv.Addr).Msg("completing background tasks")

		// Wait for all background goroutines to finish
		if s.wg != nil {
			s.wg.Wait()
		}

		s.logger.Info().Str("addr", srv.Addr).Msg("shutdown complete")
	}()

	s.logStartup()

	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	// Wait for the shutdown goroutine to finish
	<-shutdownDone

	s.logger.Info().Str("addr", srv.Addr).Msg("stopped server")

	return nil
}

// URL returns a clickable server URL for the user display.
// If the server binds to all interfaces, localhost is used.
func (s *Server) URL() string {
	displayHost := s.host
	switch displayHost {
	case "", "*", "0.0.0.0", "::":
		displayHost = "localhost"
	}

	if ip := net.ParseIP(displayHost); ip != nil && ip.To4() == nil {
		// IPv6
		return fmt.Sprintf("http://[%s]:%d/", displayHost, s.port)
	}
	// IPv4 or localhost
	return fmt.Sprintf("http://%s:%d/", displayHost, s.port)
}

// logStartup prints the startup log for the server.
// It shows `*:port` if the host is empty (all interfaces).
func (s *Server) logStartup() {
	addrLog := net.JoinHostPort(s.host, strconv.Itoa(s.port))
	if s.host == "" {
		addrLog = fmt.Sprintf("*:%d", s.port)
	}
	s.logger.Info().Str("addr", addrLog).Msg("starting server")
	s.logger.Info().Msgf("Server accessible at %s", s.URL())
}
