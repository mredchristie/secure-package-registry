package external

import (
	"bytes"
	"encoding/gob"
	"errors"
	"net/http"
	"strconv"

	"git.duti.dev/secure-package-registry/internal/gen/coredb"
	"git.duti.dev/secure-package-registry/internal/messages"
	"git.duti.dev/secure-package-registry/pkg/lockfile"
	"git.duti.dev/secure-package-registry/pkg/logger"
	"git.duti.dev/secure-package-registry/pkg/npm"
	"git.duti.dev/secure-package-registry/pkg/verification"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
)

// ProjectHandler handles project CRUD and dependency management.
type ProjectHandler struct {
	db        coredb.Querier
	publisher message.Publisher
	resolver  *npm.Resolver
	verifier  *verification.Service
	npmClient *npm.Client
	log       zerolog.Logger
}

// NewProjectHandler creates a new ProjectHandler and registers its routes.
// The returned handler must be mounted behind AuthMiddleware.
func NewProjectHandler(db coredb.Querier, publisher message.Publisher, verifier *verification.Service, npmClient *npm.Client) http.Handler {
	h := &ProjectHandler{
		db:        db,
		publisher: publisher,
		resolver:  npm.NewResolver(npmClient),
		verifier:  verifier,
		npmClient: npmClient,
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

// UploadProject accepts a project file, validates it, stores it for async
// processing, and returns 202 Accepted immediately. The actual dependency
// resolution and storage happens in the background consumer.
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

	fileBytes := []byte(req.File)

	// Quick validation: parse the file to ensure it's valid before accepting.
	parsed, err := lockfile.Parse(fileBytes)
	if err != nil {
		h.log.Warn().Err(err).Str("project", req.Name).Msg("Failed to parse uploaded file")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "failed to parse file: " + err.Error()})
		return
	}

	ctx := r.Context()

	// Upsert the project with the raw file stored for async processing.
	// On re-upload this resets status to 'pending' and bumps generation.
	project, err := h.db.InsertProject(ctx, coredb.InsertProjectParams{
		UserID:     userID,
		Name:       req.Name,
		SourceType: string(parsed.SourceType),
		SourceFile: fileBytes,
	})
	if err != nil {
		h.log.Error().Err(err).Str("project", req.Name).Msg("Failed to upsert project")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "failed to create project"})
		return
	}

	// Publish processing request for the background consumer.
	msg := messages.ProjectProcessingRequested{
		ProjectID:  project.ID,
		Generation: project.Generation,
	}
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(msg); err != nil {
		h.log.Error().Err(err).Msg("Failed to encode project processing request")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "failed to enqueue processing"})
		return
	}
	if err := h.publisher.Publish("spr.project.processing.requested", message.NewMessage(watermill.NewUUID(), buf.Bytes())); err != nil {
		h.log.Error().Err(err).Str("project", req.Name).Msg("Failed to publish project processing request")
		// The project is stored with status=pending; a retry mechanism or
		// re-upload can recover. Don't fail the request.
	}

	render.Status(r, http.StatusAccepted)
	render.JSON(w, r, UploadProjectResponse{
		ID:     project.ID,
		Name:   project.Name,
		Status: project.Status,
	})
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
			Status:     p.Status,
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
		Status:     project.Status,
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
			HasAttestation: boolPtrFromInterface(d.HasAttestation),
			HasOssRebuild:  boolPtrFromInterface(d.HasOssRebuild),
			BehaviorPassed: boolPtrFromInterface(d.BehaviorPassed),
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

// boolPtrFromInterface converts a database interface{} (bool or nil) to *bool.
// It handles the nullable boolean fields from sqlc queries.
func boolPtrFromInterface(v interface{}) *bool {
	if v == nil {
		return nil
	}
	if b, ok := v.(bool); ok {
		return &b
	}
	return nil
}
