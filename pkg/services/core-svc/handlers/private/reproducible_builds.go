package private

import (
	"encoding/base64"
	"net/http"

	"git.duti.dev/secure-package-registry/internal/gen/coredb"
	"git.duti.dev/secure-package-registry/pkg/gitea"
	"git.duti.dev/secure-package-registry/pkg/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/rs/zerolog"
)

// ReproducibleBuildHandler handles registration of reproducible build artifacts.
type ReproducibleBuildHandler struct {
	db       coredb.Querier
	registry *gitea.NpmRegistry
	log      zerolog.Logger
}

// RegisterReproducibleBuildRequest is the payload for registering a reproducible build.
type RegisterReproducibleBuildRequest struct {
	Ecosystem  string `json:"ecosystem"`
	Identifier string `json:"identifier"`
	Version    string `json:"version"`
	// Tarball is the base64-encoded .tgz artifact.
	Tarball string `json:"tarball"`
}

// RegisterReproducibleBuildResponse is returned on success.
type RegisterReproducibleBuildResponse struct {
	PackageVersionID int32 `json:"package_version_id"`
}

// NewReproducibleBuildHandler creates a chi router for reproducible build endpoints.
func NewReproducibleBuildHandler(db coredb.Querier, registry *gitea.NpmRegistry) http.Handler {
	h := &ReproducibleBuildHandler{
		db:       db,
		registry: registry,
		log:      logger.WithComponent("reproducible-builds"),
	}

	r := chi.NewRouter()
	r.Post("/", h.Register)

	return r
}

// Register handles POST /api/v1/internal/reproducible-builds.
// It validates the package version exists, tags it as reproducible, and uploads
// the artifact to the spr-registry Gitea account.
func (h *ReproducibleBuildHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterReproducibleBuildRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		h.log.Warn().Err(err).Msg("Failed to decode request body")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Ecosystem == "" || req.Identifier == "" || req.Version == "" || req.Tarball == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "missing required fields: ecosystem, identifier, version, tarball"})
		return
	}

	ecosystem := coredb.Ecosystem(req.Ecosystem)
	switch ecosystem {
	case coredb.EcosystemNpm:
		// OK — only npm is supported for now.
	default:
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "unsupported ecosystem: " + req.Ecosystem + " (only npm is supported)"})
		return
	}

	ctx := r.Context()

	// 1. Look up the package version to confirm it exists.
	pvID, err := h.db.GetPackageVersionID(ctx, coredb.GetPackageVersionIDParams{
		Ecosystem:  ecosystem,
		Identifier: req.Identifier,
		Version:    req.Version,
	})
	if err != nil {
		h.log.Warn().
			Err(err).
			Str("ecosystem", req.Ecosystem).
			Str("identifier", req.Identifier).
			Str("version", req.Version).
			Msg("Package version not found")
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "package version not found"})
		return
	}

	// 2. Look up the "reproducible" tag type.
	tagTypeID, err := h.db.GetTagTypeByLabel(ctx, "reproducible")
	if err != nil {
		h.log.Error().Err(err).Msg("Tag type 'reproducible' not found — has migration 000005 been applied?")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "reproducible tag type not configured"})
		return
	}

	// 3. Tag the package version as reproducible.
	if err := h.db.InsertPackageTag(ctx, coredb.InsertPackageTagParams{
		PackageVersion: pvID,
		TagType:        tagTypeID,
		Value:          []byte(`true`),
	}); err != nil {
		h.log.Error().Err(err).Int32("package_version_id", pvID).Msg("Failed to insert reproducible tag")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "failed to tag package version"})
		return
	}

	// 4. Decode the tarball and upload to spr-registry Gitea account.
	tarball, err := base64.StdEncoding.DecodeString(req.Tarball)
	if err != nil {
		h.log.Warn().Err(err).Msg("Invalid base64 tarball")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "tarball is not valid base64"})
		return
	}

	if err := h.registry.UploadPackage(ctx, req.Identifier, req.Version, tarball, nil); err != nil {
		h.log.Error().Err(err).
			Str("identifier", req.Identifier).
			Str("version", req.Version).
			Msg("Failed to upload artifact to Gitea registry")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "failed to upload artifact to registry"})
		return
	}

	h.log.Info().
		Str("ecosystem", req.Ecosystem).
		Str("identifier", req.Identifier).
		Str("version", req.Version).
		Int32("package_version_id", pvID).
		Msg("Registered reproducible build")

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, RegisterReproducibleBuildResponse{
		PackageVersionID: pvID,
	})
}
