// Package server provides HTTP server implementations for the core service.
package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"git.duti.dev/secure-package-registry/pkg/logger"
	"github.com/rs/zerolog"
)

// Server is a wrapper around http.Server with graceful shutdown support.
type Server struct {
	name   string
	server *http.Server
	log    zerolog.Logger
}

// New creates a new Server instance.
func New(name, addr string, handler http.Handler) *Server {
	return &Server{
		name: name,
		server: &http.Server{
			Addr:         addr,
			Handler:      handler,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
		log: logger.WithComponent("core-svc:" + name),
	}
}

// Start begins listening for requests in a goroutine.
// Returns a channel that receives any startup error.
func (s *Server) Start() chan error {
	errChan := make(chan error, 1)

	go func() {
		s.log.Info().
			Str("addr", s.server.Addr).
			Msgf("Starting %s server", s.name)

		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log.Error().
				Err(err).
				Str("addr", s.server.Addr).
				Msgf("%s server error", s.name)
			errChan <- fmt.Errorf("%s server error: %w", s.name, err)
		}
	}()

	return errChan
}

// Stop gracefully shuts down the server.
func (s *Server) Stop(ctx context.Context) error {
	s.log.Info().Msgf("Shutting down %s server", s.name)

	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("failed to shutdown %s server: %w", s.name, err)
	}

	s.log.Info().Msgf("%s server stopped", s.name)
	return nil
}
