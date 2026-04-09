package external

import (
	"net/http"
	"net/url"

	"git.duti.dev/secure-package-registry/pkg/logger"
	"git.duti.dev/secure-package-registry/pkg/verification"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/rs/zerolog"
)

// VerificationHandler handles package version verification requests.
type VerificationHandler struct {
	verifier *verification.Service
	log      zerolog.Logger
}

// NewVerificationHandler creates a VerificationHandler.
func NewVerificationHandler(verifier *verification.Service) *VerificationHandler {
	return &VerificationHandler{
		verifier: verifier,
		log:      logger.WithComponent("verification-handler"),
	}
}

// Verify re-checks upstream attestation and OSS rebuild for a package version.
func (h *VerificationHandler) Verify(w http.ResponseWriter, r *http.Request) {
	ecosystem := chi.URLParam(r, "ecosystem")
	identifier, _ := url.PathUnescape(chi.URLParam(r, "identifier"))
	version := chi.URLParam(r, "version")

	if ecosystem == "" || identifier == "" || version == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "missing required path parameters"})
		return
	}

	result, err := h.verifier.CheckAndTag(r.Context(), ecosystem, identifier, version)
	if err != nil {
		h.log.Warn().Err(err).
			Str("ecosystem", ecosystem).
			Str("identifier", identifier).
			Str("version", version).
			Msg("Verification failed")
		render.Status(r, http.StatusUnprocessableEntity)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	render.JSON(w, r, result)
}
