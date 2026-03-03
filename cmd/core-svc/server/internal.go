package server

import (
	"net/http"

	"git.duti.dev/secure-package-registry/cmd/core-svc/handlers/private"
	"git.duti.dev/secure-package-registry/internal/gen/coredb"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewInternal(addr string, db coredb.Querier, publisher message.Publisher) *Server {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Route("/api/v1/internal", func(r chi.Router) {
		r.Mount("/packages", private.NewPackageHandler(db, publisher))
	})

	return New("internal", addr, r)
}
