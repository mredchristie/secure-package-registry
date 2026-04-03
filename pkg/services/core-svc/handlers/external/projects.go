package external

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"net/http"
	"strconv"

	"git.duti.dev/secure-package-registry/internal/gen/coredb"
	"git.duti.dev/secure-package-registry/internal/messages"
	"git.duti.dev/secure-package-registry/pkg/lockfile"
	"git.duti.dev/secure-package-registry/pkg/logger"
	"git.duti.dev/secure-package-registry/pkg/verification"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
)

// maxConcurrentScans is the per-user limit for concurrent behavioral analysis runs.
const maxConcurrentScans = 5

// ProjectHandler handles project CRUD and dependency management.
type ProjectHandler struct {
	db        coredb.Querier
	publisher message.Publisher
	verifier  *verification.Service
	log       zerolog.Logger
}

// NewProjectHandler creates a new ProjectHandler and registers its routes.
// The returned handler must be mounted behind AuthMiddleware.
func NewProjectHandler(db coredb.Querier, publisher message.Publisher, verifier *verification.Service) http.Handler {
	h := &ProjectHandler{
		db:        db,
		publisher: publisher,
		verifier:  verifier,
		log:       logger.WithComponent("project-handler"),
	}

	r := chi.NewRouter()
	r.Post("/", h.UploadProject)
	r.Get("/", h.ListProjects)
	r.Get("/{projectID}", h.GetProject)
	r.Get("/{projectID}/dependencies", h.ListDependencies)
	r.Get("/{projectID}/summary", h.GetSummary)
	r.Delete("/{projectID}", h.DeleteProject)

	return r
}

// UploadProject creates or replaces a project's dependency set.
func (h *ProjectHandler) UploadProject(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	if userID == "" {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "not authenticated"})
		return
	}

	var req UploadProjectRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Name == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "missing required field: name"})
		return
	}
	if req.File == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "missing required field: file"})
		return
	}

	// Parse the uploaded file.
	parsed, err := lockfile.Parse([]byte(req.File))
	if err != nil {
		h.log.Warn().Err(err).Str("project", req.Name).Msg("Failed to parse uploaded file")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "failed to parse file: " + err.Error()})
		return
	}

	ctx := r.Context()

	// Upsert the project.
	project, err := h.db.InsertProject(ctx, coredb.InsertProjectParams{
		UserID:     userID,
		Name:       req.Name,
		SourceType: string(parsed.SourceType),
	})
	if err != nil {
		h.log.Error().Err(err).Str("project", req.Name).Msg("Failed to upsert project")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "failed to create project"})
		return
	}

	// Atomically replace deps: delete all existing, then insert new.
	if err := h.db.DeleteProjectDependencies(ctx, project.ID); err != nil {
		h.log.Error().Err(err).Int32("project_id", project.ID).Msg("Failed to clear project dependencies")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "failed to update project dependencies"})
		return
	}

	// Track how many scans we trigger for the concurrency limit.
	scansTriggered := 0

	for _, dep := range parsed.All {
		// For package.json files, Version is empty and only Constraint is set.
		// Use the constraint as the version string so we have something meaningful.
		version := dep.Version
		if version == "" {
			version = dep.Constraint
		}

		// Upsert the package.
		pkgID, err := h.db.InsertPackage(ctx, coredb.InsertPackageParams{
			Identifier:    dep.Name,
			Ecosystem:     coredb.EcosystemNpm,
			LatestVersion: pgtype.Text{},
		})
		if err != nil {
			h.log.Error().Err(err).Str("dep", dep.Name).Msg("Failed to upsert package")
			continue
		}

		// Upsert the package version.
		pvID, err := h.db.InsertPackageVersion(ctx, coredb.InsertPackageVersionParams{
			PackageID: pkgID,
			Version:   version,
			SourceUrl: pgtype.Text{},
		})
		if err != nil {
			h.log.Error().Err(err).Str("dep", dep.Name+"@"+version).Msg("Failed to upsert package version")
			continue
		}

		// Determine dependency type.
		depType := coredb.DependencyTypeTransitive
		if dep.Direct {
			depType = coredb.DependencyTypeDirect
		}

		// Insert the project dependency.
		if err := h.db.InsertProjectDependency(ctx, coredb.InsertProjectDependencyParams{
			ProjectID:        project.ID,
			PackageID:        pkgID,
			PackageVersionID: pvID,
			DependencyType:   depType,
			VersionConstraint: pgtype.Text{
				String: dep.Constraint,
				Valid:  dep.Constraint != "",
			},
		}); err != nil {
			h.log.Error().Err(err).Str("dep", dep.Name+"@"+version).Msg("Failed to insert project dependency")
			continue
		}

		// Auto-verify (attestation + OSS rebuild) for ALL deps, best-effort.
		if _, verErr := h.verifier.CheckAndTag(ctx, "npm", dep.Name, version); verErr != nil {
			h.log.Debug().Err(verErr).Str("dep", dep.Name+"@"+version).Msg("Auto-verification failed (non-fatal)")
		}

		// Trigger behavioral analysis for DIRECT deps only, respecting concurrency limit.
		if dep.Direct && scansTriggered < maxConcurrentScans {
			if triggered := h.triggerBehavioralAnalysis(ctx, dep.Name, version, pvID); triggered {
				scansTriggered++
			}
		}
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, UploadProjectResponse{
		ID:         project.ID,
		Name:       project.Name,
		SourceType: project.SourceType,
		TotalDeps:  len(parsed.All),
		DirectDeps: len(parsed.DirectDeps()),
	})
}

// triggerBehavioralAnalysis creates a collection task and publishes a scan
// request for a single package version. Returns true if a scan was triggered.
func (h *ProjectHandler) triggerBehavioralAnalysis(ctx context.Context, identifier, version string, pvID int32) bool {
	// Dedup: skip if already active.
	active, err := h.db.HasActiveCollectionTask(ctx, coredb.HasActiveCollectionTaskParams{
		PackageVersionID: pvID,
		Source:           "npm",
	})
	if err != nil {
		h.log.Error().Err(err).Str("dep", identifier+"@"+version).Msg("Failed to check active collection task")
		return false
	}
	if active {
		return false
	}

	task, err := h.db.InsertCollectionTask(ctx, coredb.InsertCollectionTaskParams{
		PackageVersionID: pvID,
		Source:           "npm",
	})
	if errors.Is(err, pgx.ErrNoRows) {
		// Conflict — task already exists.
		return false
	}
	if err != nil {
		h.log.Error().Err(err).Str("dep", identifier+"@"+version).Msg("Failed to insert collection task")
		return false
	}

	collReq := messages.CollectionRequested{
		TaskID:     task.ID,
		Ecosystem:  "npm",
		Identifier: identifier,
		Version:    version,
	}
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(collReq); err != nil {
		h.log.Error().Err(err).Msg("Failed to encode collection request")
		return true // task was created even if publish failed
	}
	if err := h.publisher.Publish("spr.collection.requested", message.NewMessage(watermill.NewUUID(), buf.Bytes())); err != nil {
		h.log.Error().Err(err).Str("dep", identifier+"@"+version).Msg("Failed to publish collection request")
	}

	h.log.Info().Str("dep", identifier+"@"+version).Int32("task_id", task.ID).Msg("Triggered behavioral analysis")
	return true
}

// ListProjects returns all projects belonging to the authenticated user.
func (h *ProjectHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	if userID == "" {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "not authenticated"})
		return
	}

	projects, err := h.db.ListUserProjects(r.Context(), userID)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to list projects")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "failed to list projects"})
		return
	}

	items := make([]ProjectListItem, 0, len(projects))
	for _, p := range projects {
		item := ProjectListItem{
			ID:         p.ID,
			Name:       p.Name,
			SourceType: p.SourceType,
		}
		if p.CreatedAt.Valid {
			item.CreatedAt = p.CreatedAt.Time.String()
		}
		if p.UpdatedAt.Valid {
			item.UpdatedAt = p.UpdatedAt.Time.String()
		}
		items = append(items, item)
	}

	render.JSON(w, r, ProjectListResponse{Items: items})
}

// GetProject returns a single project by ID, verifying ownership.
func (h *ProjectHandler) GetProject(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	if userID == "" {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "not authenticated"})
		return
	}

	projectID, err := strconv.Atoi(chi.URLParam(r, "projectID"))
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "invalid project ID"})
		return
	}

	project, err := h.db.GetProject(r.Context(), int32(projectID))
	if errors.Is(err, pgx.ErrNoRows) {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "project not found"})
		return
	}
	if err != nil {
		h.log.Error().Err(err).Int("project_id", projectID).Msg("Failed to get project")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "failed to get project"})
		return
	}

	if project.UserID != userID {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "project not found"})
		return
	}

	resp := GetProjectResponse{
		ID:         project.ID,
		Name:       project.Name,
		SourceType: project.SourceType,
	}
	if project.CreatedAt.Valid {
		resp.CreatedAt = project.CreatedAt.Time.String()
	}
	if project.UpdatedAt.Valid {
		resp.UpdatedAt = project.UpdatedAt.Time.String()
	}

	render.JSON(w, r, resp)
}

// ListDependencies returns dependencies for a project, optionally filtered by type.
func (h *ProjectHandler) ListDependencies(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	if userID == "" {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "not authenticated"})
		return
	}

	projectID, err := strconv.Atoi(chi.URLParam(r, "projectID"))
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "invalid project ID"})
		return
	}

	// Verify ownership.
	project, err := h.db.GetProject(r.Context(), int32(projectID))
	if errors.Is(err, pgx.ErrNoRows) {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "project not found"})
		return
	}
	if err != nil {
		h.log.Error().Err(err).Int("project_id", projectID).Msg("Failed to get project")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "failed to get project"})
		return
	}
	if project.UserID != userID {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "project not found"})
		return
	}

	// Build optional type filter.
	var depType coredb.NullDependencyType
	if typeStr := r.URL.Query().Get("type"); typeStr != "" {
		dt := coredb.DependencyType(typeStr)
		if dt != coredb.DependencyTypeDirect && dt != coredb.DependencyTypeTransitive {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, map[string]string{"error": "invalid type filter: must be 'direct' or 'transitive'"})
			return
		}
		depType = coredb.NullDependencyType{DependencyType: dt, Valid: true}
	}

	deps, err := h.db.ListProjectDependencies(r.Context(), coredb.ListProjectDependenciesParams{
		ProjectID: int32(projectID),
		DepType:   depType,
	})
	if err != nil {
		h.log.Error().Err(err).Int("project_id", projectID).Msg("Failed to list dependencies")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "failed to list dependencies"})
		return
	}

	items := make([]DependencyListItem, 0, len(deps))
	for _, d := range deps {
		item := DependencyListItem{
			ID:             d.ID,
			Identifier:     d.Identifier,
			Ecosystem:      d.PEcosystem,
			Version:        d.Version,
			DependencyType: string(d.DependencyType),
		}
		if d.VersionConstraint.Valid {
			item.VersionConstraint = d.VersionConstraint.String
		}
		items = append(items, item)
	}

	render.JSON(w, r, DependencyListResponse{Items: items})
}

// GetSummary returns aggregated security posture stats for a project.
func (h *ProjectHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	if userID == "" {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "not authenticated"})
		return
	}

	projectID, err := strconv.Atoi(chi.URLParam(r, "projectID"))
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "invalid project ID"})
		return
	}

	// Verify ownership.
	project, err := h.db.GetProject(r.Context(), int32(projectID))
	if errors.Is(err, pgx.ErrNoRows) {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "project not found"})
		return
	}
	if err != nil {
		h.log.Error().Err(err).Int("project_id", projectID).Msg("Failed to get project")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "failed to get project"})
		return
	}
	if project.UserID != userID {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "project not found"})
		return
	}

	rows, err := h.db.GetProjectSummary(r.Context(), int32(projectID))
	if err != nil {
		h.log.Error().Err(err).Int("project_id", projectID).Msg("Failed to get project summary")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "failed to get project summary"})
		return
	}

	items := make([]SummaryRow, 0, len(rows))
	for _, row := range rows {
		items = append(items, SummaryRow{
			DependencyType: string(row.DependencyType),
			Total:          row.Total,
			HasAttestation: row.HasAttestation,
			HasOssRebuild:  row.HasOssRebuild,
			BehaviorPassed: row.BehaviorPassed,
		})
	}

	render.JSON(w, r, ProjectSummaryResponse{
		ProjectID: projectID,
		Summary:   items,
	})
}

// DeleteProject removes a project and all its dependencies.
func (h *ProjectHandler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	if userID == "" {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "not authenticated"})
		return
	}

	projectID, err := strconv.Atoi(chi.URLParam(r, "projectID"))
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "invalid project ID"})
		return
	}

	// DeleteProject enforces ownership via WHERE id = $1 AND user_id = $2.
	if err := h.db.DeleteProject(r.Context(), coredb.DeleteProjectParams{
		ID:     int32(projectID),
		UserID: userID,
	}); err != nil {
		h.log.Error().Err(err).Int("project_id", projectID).Msg("Failed to delete project")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "failed to delete project"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
