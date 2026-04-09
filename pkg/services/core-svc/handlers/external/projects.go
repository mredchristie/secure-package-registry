package external

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"net/http"
	"strconv"
	"strings"

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
	"github.com/jackc/pgx/v5/pgtype"
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

	// --- Phase 1: Resolve direct dep constraints (package.json only) ---
	var directDeps []lockfile.Dep
	skipped := 0
	for _, dep := range parsed.DirectDeps() {
		if isNonStandardSpecifier(dep.Constraint) {
			h.log.Warn().
				Str("dep", dep.Name).
				Str("specifier", dep.Constraint).
				Msg("Skipping non-standard dependency specifier")
			skipped++
			continue
		}

		version := dep.Version
		if version == "" && dep.Constraint != "" {
			resolved, err := h.npmClient.ResolveConstraint(ctx, dep.Name, dep.Constraint)
			if err != nil {
				h.log.Warn().Err(err).
					Str("dep", dep.Name).
					Str("constraint", dep.Constraint).
					Msg("Failed to resolve version constraint — skipping dependency")
				skipped++
				continue
			}
			version = resolved
		}
		if version == "" {
			h.log.Warn().Str("dep", dep.Name).Msg("No version or constraint — skipping dependency")
			skipped++
			continue
		}

		directDeps = append(directDeps, lockfile.Dep{
			Name:       dep.Name,
			Version:    version,
			Constraint: dep.Constraint,
			Direct:     true,
		})
	}

	// --- Phase 2: Expand transitive tree ---
	var allDeps []lockfile.Dep
	if parsed.SourceType == lockfile.SourcePackageJSON {
		transitiveDeps := h.resolveTransitiveDeps(ctx, directDeps)
		allDeps = make([]lockfile.Dep, 0, len(directDeps)+len(transitiveDeps))
		allDeps = append(allDeps, directDeps...)
		allDeps = append(allDeps, transitiveDeps...)
	} else {
		// Lock file already has the complete dependency set.
		allDeps = parsed.All
	}

	// --- Phase 3: Store and trigger analysis ---
	for _, dep := range allDeps {
		version := dep.Version
		if version == "" && dep.Constraint != "" {
			resolved, err := h.npmClient.ResolveConstraint(ctx, dep.Name, dep.Constraint)
			if err != nil {
				h.log.Warn().Err(err).
					Str("dep", dep.Name).
					Str("constraint", dep.Constraint).
					Msg("Failed to resolve version constraint — skipping dependency")
				continue
			}
			version = resolved
		}
		if version == "" {
			h.log.Warn().Str("dep", dep.Name).Msg("No version or constraint — skipping dependency")
			continue
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

		// Trigger behavioral analysis for DIRECT deps only.
		if dep.Direct {
			h.triggerBehavioralAnalysis(ctx, dep.Name, version, pvID)
		}
	}

	totalDeps := len(allDeps)
	directCount := len(directDeps)
	if parsed.SourceType != lockfile.SourcePackageJSON {
		directCount = len(parsed.DirectDeps())
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, UploadProjectResponse{
		ID:             project.ID,
		Name:           project.Name,
		SourceType:     project.SourceType,
		TotalDeps:      totalDeps,
		DirectDeps:     directCount,
		TransitiveDeps: totalDeps - directCount,
		Skipped:        skipped,
	})
}

// isNonStandardSpecifier returns true if the constraint string is a git URL,
// GitHub shorthand, tarball URL, or local file path rather than a semver range.
// These are skipped during resolution (see RFC 2026-04-09).
func isNonStandardSpecifier(constraint string) bool {
	if constraint == "" {
		return false
	}

	// Git protocols.
	for _, prefix := range []string{"git+", "git://", "git@"} {
		if strings.HasPrefix(constraint, prefix) {
			return true
		}
	}

	// Tarball URLs.
	if strings.HasPrefix(constraint, "http://") || strings.HasPrefix(constraint, "https://") {
		return true
	}

	// Local file paths.
	if strings.HasPrefix(constraint, "file:") {
		return true
	}

	// GitHub shorthand: "user/repo" or "user/repo#ref".
	// Must contain exactly one "/" with no spaces and no semver-range chars.
	if strings.Contains(constraint, "/") &&
		!strings.ContainsAny(constraint, " <>!=^~*|") {
		parts := strings.SplitN(constraint, "#", 2)
		segments := strings.Split(parts[0], "/")
		if len(segments) == 2 && segments[0] != "" && segments[1] != "" {
			return true
		}
	}

	return false
}

// resolveTransitiveDeps resolves the full transitive dependency tree for each
// direct dependency using the npm registry resolver. Returns only the transitive
// deps (direct deps are excluded since they are already in the caller's list).
func (h *ProjectHandler) resolveTransitiveDeps(ctx context.Context, directDeps []lockfile.Dep) []lockfile.Dep {
	// Pre-populate seen set with direct deps to avoid duplicates and ensure
	// direct type takes precedence.
	seen := make(map[string]bool, len(directDeps))
	for _, d := range directDeps {
		seen[d.Name+"@"+d.Version] = true
	}

	var transitive []lockfile.Dep

	for _, dep := range directDeps {
		graph, err := h.resolver.Resolve(ctx, dep.Name, dep.Version)
		if err != nil {
			h.log.Warn().Err(err).
				Str("dep", dep.Name+"@"+dep.Version).
				Msg("Failed to resolve transitive deps — skipping tree")
			continue
		}

		for key, node := range graph.Nodes {
			if key == graph.Root {
				continue // skip the direct dep itself
			}
			if node == nil {
				continue // incomplete resolution (in-progress placeholder)
			}
			if seen[key] {
				continue
			}
			seen[key] = true

			transitive = append(transitive, lockfile.Dep{
				Name:    node.Name,
				Version: node.Version,
				Direct:  false,
			})
		}
	}

	h.log.Info().
		Int("direct", len(directDeps)).
		Int("transitive", len(transitive)).
		Msg("Resolved transitive dependency tree")

	return transitive
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
