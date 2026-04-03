// Package external provides handlers for the external API.
package external

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"git.duti.dev/secure-package-registry/internal/gen/coredb"
	"git.duti.dev/secure-package-registry/internal/messages"
	"git.duti.dev/secure-package-registry/pkg/logger"
	sprminio "git.duti.dev/secure-package-registry/pkg/minio"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
)

// AdminHandler handles admin routes for package and task management.
type AdminHandler struct {
	db        coredb.Querier
	publisher message.Publisher
	minio     *sprminio.Client
	log       zerolog.Logger
}

// NewAdminHandler creates a new AdminHandler and registers its routes.
func NewAdminHandler(db coredb.Querier, publisher message.Publisher, minio *sprminio.Client) http.Handler {
	h := &AdminHandler{
		db:        db,
		publisher: publisher,
		minio:     minio,
		log:       logger.WithComponent("admin-handler"),
	}

	r := chi.NewRouter()
	r.Get("/packages", h.ListPackages)
	r.Post("/packages", h.AddPackage)
	r.Get("/packages/{ecosystem}/{identifier}/versions", h.ListVersions)
	r.Post("/packages/{ecosystem}/{identifier}/scan", h.TriggerScan)
	r.Get("/packages/{ecosystem}/{identifier}/behavior", h.GetBehavior)
	r.Get("/packages/{ecosystem}/{identifier}/behavior/raw", h.GetBehaviorRaw)
	r.Get("/tasks", h.ListTasks)
	r.Get("/tasks/{taskID}/artifact", h.DownloadArtifact)

	return r
}

// ListPackages returns all watched packages for a given ecosystem.
func (h *AdminHandler) ListPackages(w http.ResponseWriter, r *http.Request) {
	ecoStr := r.URL.Query().Get("ecosystem")
	if ecoStr == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "missing required parameter: ecosystem"})
		return
	}

	ecosystem := coredb.Ecosystem(ecoStr)
	if !validEcosystem(ecosystem) {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "invalid ecosystem: " + ecoStr})
		return
	}

	packages, err := h.db.ListPackagesByEcosystem(r.Context(), ecosystem)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to list packages")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "failed to list packages"})
		return
	}

	items := make([]PackageListItem, 0, len(packages))
	for _, pkg := range packages {
		item := PackageListItem{
			ID:         pkg.ID,
			Identifier: pkg.Identifier,
			Ecosystem:  string(pkg.Ecosystem),
		}
		if pkg.LatestVersion.Valid {
			item.LatestVersion = &pkg.LatestVersion.String
		}
		items = append(items, item)
	}

	render.JSON(w, r, PackageListResponse{Items: items})
}

// AddPackage creates a new package in the watch list and publishes spr.package.requested.
func (h *AdminHandler) AddPackage(w http.ResponseWriter, r *http.Request) {
	var req AddPackageRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
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
	if !validEcosystem(ecosystem) {
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
		h.log.Error().Err(err).Str("identifier", req.Identifier).Msg("Failed to insert package")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "failed to create package"})
		return
	}

	// Publish spr.package.requested so the poller picks it up.
	if pubErr := h.publishPackageRequested(req.Identifier, req.Ecosystem); pubErr != nil {
		h.log.Error().Err(pubErr).Str("identifier", req.Identifier).Msg("Failed to publish package requested event")
	}

	status := http.StatusCreated
	if alreadyExists {
		status = http.StatusOK
	}

	render.Status(r, status)
	render.JSON(w, r, AddPackageResponse{
		ID:            packageID,
		Identifier:    req.Identifier,
		Ecosystem:     req.Ecosystem,
		AlreadyExists: alreadyExists,
	})
}

// ListVersions returns all known versions for a package.
func (h *AdminHandler) ListVersions(w http.ResponseWriter, r *http.Request) {
	ecoStr := chi.URLParam(r, "ecosystem")
	identifier, _ := url.PathUnescape(chi.URLParam(r, "identifier"))

	ecosystem := coredb.Ecosystem(ecoStr)
	if !validEcosystem(ecosystem) {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "invalid ecosystem: " + ecoStr})
		return
	}

	ctx := r.Context()

	pkg, err := h.db.GetPackageByEcosystemAndIdentifier(ctx, coredb.GetPackageByEcosystemAndIdentifierParams{
		Ecosystem:  ecosystem,
		Identifier: identifier,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "package not found"})
		return
	}
	if err != nil {
		h.log.Error().Err(err).Str("identifier", identifier).Msg("Failed to look up package")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "failed to look up package"})
		return
	}

	versions, err := h.db.ListPackageVersions(ctx, pkg.ID)
	if err != nil {
		h.log.Error().Err(err).Str("identifier", identifier).Msg("Failed to list versions")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "failed to list versions"})
		return
	}

	var latestVersion *string
	if pkg.LatestVersion.Valid {
		latestVersion = &pkg.LatestVersion.String
	}

	render.JSON(w, r, ListVersionsResponse{
		Identifier:    identifier,
		Ecosystem:     ecoStr,
		LatestVersion: latestVersion,
		Versions:      versions,
	})
}

// TriggerScan triggers a behavioral analysis scan for a package version.
// If no version is specified, defaults to the stored latest_version.
func (h *AdminHandler) TriggerScan(w http.ResponseWriter, r *http.Request) {
	ecoStr := chi.URLParam(r, "ecosystem")
	identifier, _ := url.PathUnescape(chi.URLParam(r, "identifier"))

	ecosystem := coredb.Ecosystem(ecoStr)
	if !validEcosystem(ecosystem) {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "invalid ecosystem: " + ecoStr})
		return
	}

	ctx := r.Context()

	// Parse optional version from body.
	var req TriggerScanRequest
	// Body may be empty; ignore decode errors for an empty body.
	_ = render.DecodeJSON(r.Body, &req)

	// Look up the package.
	pkg, err := h.db.GetPackageByEcosystemAndIdentifier(ctx, coredb.GetPackageByEcosystemAndIdentifierParams{
		Ecosystem:  ecosystem,
		Identifier: identifier,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "package not found"})
		return
	}
	if err != nil {
		h.log.Error().Err(err).Str("identifier", identifier).Msg("Failed to look up package")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "failed to look up package"})
		return
	}

	// Determine which version to scan.
	version := req.Version
	if version == "" {
		if !pkg.LatestVersion.Valid {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, map[string]string{"error": "no version specified and package has no known latest version"})
			return
		}
		version = pkg.LatestVersion.String
	}

	// Upsert the package version (it may not exist yet if manually triggered).
	pvID, err := h.db.InsertPackageVersion(ctx, coredb.InsertPackageVersionParams{
		PackageID: pkg.ID,
		Version:   version,
		SourceUrl: pgtype.Text{}, // null
	})
	if err != nil {
		h.log.Error().Err(err).Str("identifier", identifier).Str("version", version).Msg("Failed to upsert package version")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "failed to upsert package version"})
		return
	}

	// Insert collection task. ON CONFLICT DO NOTHING handles dedup.
	task, err := h.db.InsertCollectionTask(ctx, coredb.InsertCollectionTaskParams{
		PackageVersionID: pvID,
		Source:           "npm",
	})
	if errors.Is(err, pgx.ErrNoRows) {
		// Task already exists. Check if it failed/cancelled and can be reset.
		render.Status(r, http.StatusConflict)
		render.JSON(w, r, map[string]string{"error": "a collection task already exists for this version"})
		return
	}
	if err != nil {
		h.log.Error().Err(err).Str("identifier", identifier).Str("version", version).Msg("Failed to insert collection task")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "failed to create collection task"})
		return
	}

	// Publish spr.collection.requested for be-runner.
	collReq := messages.CollectionRequested{
		TaskID:     task.ID,
		Ecosystem:  ecoStr,
		Identifier: identifier,
		Version:    version,
	}
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(collReq); err != nil {
		h.log.Error().Err(err).Msg("Failed to encode collection request")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "failed to encode collection request"})
		return
	}
	if err := h.publisher.Publish("spr.collection.requested", message.NewMessage(watermill.NewUUID(), buf.Bytes())); err != nil {
		h.log.Error().Err(err).Msg("Failed to publish collection request")
		// Task is persisted, so we continue and report it was created.
	}

	h.log.Info().
		Str("package", identifier).
		Str("version", version).
		Int32("task_id", task.ID).
		Msg("Triggered scan via admin API")

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, TriggerScanResponse{
		TaskID:     task.ID,
		Identifier: identifier,
		Ecosystem:  ecoStr,
		Version:    version,
		Status:     "pending",
	})
}

// GetBehavior returns the pre-computed deduped behavioral analysis tree
// for a specific package version. The deduped tree is computed at collection
// time by the be-runner and stored as JSON in MinIO alongside the raw JSONL.
func (h *AdminHandler) GetBehavior(w http.ResponseWriter, r *http.Request) {
	ecoStr := chi.URLParam(r, "ecosystem")
	identifier, _ := url.PathUnescape(chi.URLParam(r, "identifier"))

	ecosystem := coredb.Ecosystem(ecoStr)
	if !validEcosystem(ecosystem) {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "invalid ecosystem: " + ecoStr})
		return
	}

	version := r.URL.Query().Get("version")
	if version == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "missing required parameter: version"})
		return
	}

	ctx := r.Context()

	// Find the succeeded collection task for this package version.
	task, err := h.db.GetSucceededCollectionTask(ctx, coredb.GetSucceededCollectionTaskParams{
		Ecosystem:  ecosystem,
		Identifier: identifier,
		Version:    version,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "no completed behavioral analysis found for this version"})
		return
	}
	if err != nil {
		h.log.Error().Err(err).Str("identifier", identifier).Str("version", version).Msg("Failed to look up collection task")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "failed to look up behavioral analysis"})
		return
	}

	// Derive the deduped JSON key from the raw artifact key.
	// Raw: behavior/{eco}/{pkg}/{ver}/{src}/behavior.jsonl
	// Deduped: behavior/{eco}/{pkg}/{ver}/{src}/behavior-deduped.json
	rawKey := task.ArtifactKey.String
	dedupedKey := strings.TrimSuffix(rawKey, "behavior.jsonl") + "behavior-deduped.json"

	data, err := h.minio.GetObject(ctx, dedupedKey)
	if err != nil {
		// Fall back: deduped tree may not exist yet (e.g., task completed
		// before the dedup step was added). Return 404 with a clear message.
		h.log.Warn().Err(err).Str("key", dedupedKey).Msg("Deduped behavior tree not found in MinIO")
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "behavioral analysis data not yet processed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	if _, err := io.Copy(w, bytes.NewReader(data)); err != nil {
		h.log.Error().Err(err).Msg("Failed to write behavior response")
	}
}

// GetBehaviorRaw returns the pre-computed raw (non-deduped) behavioral
// analysis tree for a specific package version.
func (h *AdminHandler) GetBehaviorRaw(w http.ResponseWriter, r *http.Request) {
	ecoStr := chi.URLParam(r, "ecosystem")
	identifier, _ := url.PathUnescape(chi.URLParam(r, "identifier"))

	ecosystem := coredb.Ecosystem(ecoStr)
	if !validEcosystem(ecosystem) {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "invalid ecosystem: " + ecoStr})
		return
	}

	version := r.URL.Query().Get("version")
	if version == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "missing required parameter: version"})
		return
	}

	ctx := r.Context()

	task, err := h.db.GetSucceededCollectionTask(ctx, coredb.GetSucceededCollectionTaskParams{
		Ecosystem:  ecosystem,
		Identifier: identifier,
		Version:    version,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "no completed behavioral analysis found for this version"})
		return
	}
	if err != nil {
		h.log.Error().Err(err).Str("identifier", identifier).Str("version", version).Msg("Failed to look up collection task")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "failed to look up behavioral analysis"})
		return
	}

	// Derive the raw JSON key from the raw artifact key.
	// Raw JSONL: behavior/{eco}/{pkg}/{ver}/{src}/behavior.jsonl
	// Raw tree:  behavior/{eco}/{pkg}/{ver}/{src}/behavior-raw.json
	rawKey := strings.TrimSuffix(task.ArtifactKey.String, "behavior.jsonl") + "behavior-raw.json"

	data, err := h.minio.GetObject(ctx, rawKey)
	if err != nil {
		h.log.Warn().Err(err).Str("key", rawKey).Msg("Raw behavior tree not found in MinIO")
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "raw behavioral analysis data not available"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	if _, err := io.Copy(w, bytes.NewReader(data)); err != nil {
		h.log.Error().Err(err).Msg("Failed to write raw behavior response")
	}
}

// ListTasks returns collection tasks with package context.
func (h *AdminHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	var ecosystem coredb.NullEcosystem
	if ecoStr := r.URL.Query().Get("ecosystem"); ecoStr != "" {
		eco := coredb.Ecosystem(ecoStr)
		if !validEcosystem(eco) {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, map[string]string{"error": "invalid ecosystem: " + ecoStr})
			return
		}
		ecosystem = coredb.NullEcosystem{Ecosystem: eco, Valid: true}
	}

	page := int32(1)
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = int32(p)
		}
	}

	pageSize := int32(50)
	if pageSizeStr := r.URL.Query().Get("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = int32(ps)
		}
	}

	tasks, err := h.db.ListCollectionTasks(r.Context(), coredb.ListCollectionTasksParams{
		Ecosystem: ecosystem,
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to list collection tasks")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "failed to list tasks"})
		return
	}

	items := make([]TaskListItem, 0, len(tasks))
	for _, t := range tasks {
		item := TaskListItem{
			ID:          t.ID,
			Identifier:  t.Identifier,
			Ecosystem:   t.PEcosystem,
			Version:     t.Version,
			Source:      t.Source,
			Status:      string(t.Status),
			HasArtifact: t.ArtifactBucket.Valid && t.ArtifactKey.Valid,
		}
		if t.FailureReason.Valid {
			item.FailureReason = &t.FailureReason.String
		}
		if t.StartedAt.Valid {
			s := t.StartedAt.Time.String()
			item.StartedAt = &s
		}
		if t.CompletedAt.Valid {
			s := t.CompletedAt.Time.String()
			item.CompletedAt = &s
		}
		if t.CreatedAt.Valid {
			item.CreatedAt = t.CreatedAt.Time.String()
		}
		items = append(items, item)
	}

	render.JSON(w, r, TaskListResponse{Items: items})
}

// DownloadArtifact streams the behavioral analysis artifact (.jsonl) for a
// completed collection task directly from MinIO.
func (h *AdminHandler) DownloadArtifact(w http.ResponseWriter, r *http.Request) {
	taskIDStr := chi.URLParam(r, "taskID")
	taskID, err := strconv.Atoi(taskIDStr)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "invalid task ID"})
		return
	}

	task, err := h.db.GetCollectionTask(r.Context(), int32(taskID))
	if errors.Is(err, pgx.ErrNoRows) {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "task not found"})
		return
	}
	if err != nil {
		h.log.Error().Err(err).Int("task_id", taskID).Msg("Failed to get collection task")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "failed to get task"})
		return
	}

	if !task.ArtifactBucket.Valid || !task.ArtifactKey.Valid {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "task has no artifact"})
		return
	}

	data, err := h.minio.GetObject(r.Context(), task.ArtifactKey.String)
	if err != nil {
		h.log.Error().Err(err).
			Str("bucket", task.ArtifactBucket.String).
			Str("key", task.ArtifactKey.String).
			Msg("Failed to download artifact from MinIO")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "failed to download artifact"})
		return
	}

	// Suggest a filename based on the key.
	filename := fmt.Sprintf("task-%d-behavior.jsonl", taskID)
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	if _, err := io.Copy(w, bytes.NewReader(data)); err != nil {
		h.log.Error().Err(err).Int("task_id", taskID).Msg("Failed to write artifact response")
	}
}

func (h *AdminHandler) publishPackageRequested(identifier, ecosystem string) error {
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

func validEcosystem(eco coredb.Ecosystem) bool {
	switch eco {
	case coredb.EcosystemNpm, coredb.EcosystemGo, coredb.EcosystemCargo, coredb.EcosystemPypi:
		return true
	default:
		return false
	}
}
