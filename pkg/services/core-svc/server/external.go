// Package server provides HTTP server implementations for the core service.
package server

import (
	"net/http"

	"git.duti.dev/secure-package-registry/internal/gen/coredb"
	sprminio "git.duti.dev/secure-package-registry/pkg/minio"
	"git.duti.dev/secure-package-registry/pkg/pkgdb"
	"git.duti.dev/secure-package-registry/pkg/services/core-svc/handlers/external"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// AdminDeps bundles the dependencies needed by admin API handlers.
type AdminDeps struct {
	Querier   coredb.Querier
	Publisher message.Publisher
	MinIO     *sprminio.Client
}

// NewExternal creates the external API server with public and admin routes.
func NewExternal(addr string, db *pkgdb.Client, admin AdminDeps) *Server {
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

	// Admin routes
	r.Mount("/api/v1/admin", external.NewAdminHandler(admin.Querier, admin.Publisher, admin.MinIO))

	return New("external", addr, r)
}
