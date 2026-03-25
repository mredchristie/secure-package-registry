package server

import (
	"net/http"

	"git.duti.dev/secure-package-registry/internal/gen/coredb"
	"git.duti.dev/secure-package-registry/pkg/gitea"
	"git.duti.dev/secure-package-registry/pkg/services/core-svc/handlers/private"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// InternalDeps bundles the dependencies needed by internal API handlers.
type InternalDeps struct {
	Querier         coredb.Querier
	Publisher       message.Publisher
	RegistryAccount *gitea.NpmRegistry
}

func NewInternal(addr string, deps InternalDeps) *Server {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Route("/api/v1/internal", func(r chi.Router) {
		r.Mount("/packages", private.NewPackageHandler(deps.Querier, deps.Publisher))
		r.Mount("/reproducible-builds", private.NewReproducibleBuildHandler(deps.Querier, deps.RegistryAccount))
	})

	return New("internal", addr, r)
}
