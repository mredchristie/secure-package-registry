// Package server provides HTTP server implementations for the core service.
package server

import (
	"net/http"

	"git.duti.dev/secure-package-registry/cmd/core-svc/handlers/external"
	"git.duti.dev/secure-package-registry/pkg/pkgdb"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewExternal creates the external API server with public routes.
func NewExternal(addr string, db *pkgdb.Client) *Server {
	r := chi.NewRouter()

	// Base middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.StripSlashes)

	// Health check (for external load balancer)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// API routes
	r.Route("/api/v1/svc", func(r chi.Router) {
		r.Mount("/packages", external.NewPackageHandler(db))
	})

	return New("external", addr, r)
}
