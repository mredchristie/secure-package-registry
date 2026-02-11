// Package server provides HTTP server implementations for the core service.
package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewInternal creates the internal API server with private routes.
func NewInternal(addr string) *Server {
	r := chi.NewRouter()

	// Minimal middleware for internal endpoints
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	// Health check (for internal monitoring)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	return New("internal", addr, r)
}
