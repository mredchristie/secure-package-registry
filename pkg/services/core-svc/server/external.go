// Package server provides HTTP server implementations for the core service.
package server

import (
	"net/http"

	"git.duti.dev/secure-package-registry/internal/gen/coredb"
	sprminio "git.duti.dev/secure-package-registry/pkg/minio"
	"git.duti.dev/secure-package-registry/pkg/pkgdb"
	"git.duti.dev/secure-package-registry/pkg/services/core-svc/handlers/external"
	"git.duti.dev/secure-package-registry/pkg/verification"
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

// ProjectDeps bundles the dependencies needed by project API handlers.
type ProjectDeps struct {
	Querier   coredb.Querier
	Publisher message.Publisher
}

// NewExternal creates the external API server with public, admin, and project routes.
func NewExternal(addr string, db *pkgdb.Client, admin AdminDeps, project ProjectDeps, verifier *verification.Service) *Server {
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
	r.Route("/api/v1/svc/packages", func(r chi.Router) {
		r.Mount("/", external.NewPackageHandler(db))
		vh := external.NewVerificationHandler(verifier)
		r.Post("/{ecosystem}/{identifier}/{version}/verify", vh.Verify)
	})

	// Admin routes
	r.Mount("/api/v1/admin", external.NewAdminHandler(admin.Querier, admin.Publisher, admin.MinIO))

	// Project routes (authenticated via API key)
	r.Route("/api/v1/projects", func(r chi.Router) {
		r.Use(external.AuthMiddleware(project.Querier))
		r.Mount("/", external.NewProjectHandler(project.Querier, project.Publisher, verifier))
	})

	return New("external", addr, r)
}
