package private

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"net/http"

	"git.duti.dev/secure-package-registry/internal/gen/coredb"
	"git.duti.dev/secure-package-registry/internal/messages"
	"git.duti.dev/secure-package-registry/pkg/logger"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
)

type PackageHandler struct {
	db        coredb.Querier
	publisher message.Publisher
	log       zerolog.Logger
}

type CreatePackageRequest struct {
	Identifier string `json:"identifier"`
	Ecosystem  string `json:"ecosystem"`
}

type CreatePackageResponse struct {
	ID            int32  `json:"id"`
	Identifier    string `json:"identifier"`
	Ecosystem     string `json:"ecosystem"`
	AlreadyExists bool   `json:"already_exists"`
}

func NewPackageHandler(db coredb.Querier, publisher message.Publisher) http.Handler {
	h := &PackageHandler{
		db:        db,
		publisher: publisher,
		log:       logger.WithComponent("internal-handler"),
	}

	r := chi.NewRouter()
	r.Post("/", h.CreatePackage)

	return r
}

func (h *PackageHandler) CreatePackage(w http.ResponseWriter, r *http.Request) {
	var req CreatePackageRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		h.log.Warn().Err(err).Msg("Failed to decode request body")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Identifier == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "missing required field: identifier"})
		return
	}

	if req.Ecosystem == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "missing required field: ecosystem"})
		return
	}

	ecosystem := coredb.Ecosystem(req.Ecosystem)
	switch ecosystem {
	case coredb.EcosystemNpm, coredb.EcosystemGo, coredb.EcosystemCargo, coredb.EcosystemPypi:
	default:
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "invalid ecosystem: " + req.Ecosystem})
		return
	}

	ctx := r.Context()

	packageID, err := h.db.InsertPackage(ctx, coredb.InsertPackageParams{
		Identifier:    req.Identifier,
		Ecosystem:     ecosystem,
		LatestVersion: pgtype.Text{Valid: false},
	})

	alreadyExists := false
	if err != nil {
		if isDuplicateKeyError(err) {
			existingPkg, getErr := h.getExistingPackage(ctx, ecosystem, req.Identifier)
			if getErr != nil {
				h.log.Error().Err(getErr).Str("identifier", req.Identifier).Msg("Failed to get existing package")
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, map[string]string{"error": "failed to check existing package"})
				return
			}
			packageID = existingPkg.ID
			alreadyExists = true
			h.log.Info().
				Int32("id", packageID).
				Str("identifier", req.Identifier).
				Msg("Package already exists")
		} else {
			h.log.Error().Err(err).Str("identifier", req.Identifier).Msg("Failed to insert package")
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, map[string]string{"error": "failed to create package"})
			return
		}
	}

	if err := h.publishPackageRequested(req.Identifier, req.Ecosystem); err != nil {
		h.log.Error().
			Err(err).
			Int32("id", packageID).
			Str("identifier", req.Identifier).
			Msg("Failed to publish package requested event")
	}

	status := http.StatusCreated
	if alreadyExists {
		status = http.StatusOK
	}

	render.Status(r, status)
	render.JSON(w, r, CreatePackageResponse{
		ID:            packageID,
		Identifier:    req.Identifier,
		Ecosystem:     req.Ecosystem,
		AlreadyExists: alreadyExists,
	})
}

func (h *PackageHandler) getExistingPackage(ctx context.Context, ecosystem coredb.Ecosystem, identifier string) (*coredb.Package, error) {
	packages, err := h.db.ListPackagesByEcosystem(ctx, ecosystem)
	if err != nil {
		return nil, err
	}

	for _, pkg := range packages {
		if pkg.Identifier == identifier {
			return &coredb.Package{
				ID:            pkg.ID,
				Identifier:    pkg.Identifier,
				Ecosystem:     pkg.Ecosystem,
				LatestVersion: pkg.LatestVersion,
			}, nil
		}
	}

	return nil, errors.New("package not found")
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

func (h *PackageHandler) publishPackageRequested(identifier, ecosystem string) error {
	req := messages.PackageRequest{
		Ecosystem:  ecosystem,
		Identifier: identifier,
	}

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(req); err != nil {
		return err
	}

	return h.publisher.Publish("spr.package.requested", message.NewMessage(watermill.NewUUID(), buf.Bytes()))
}
