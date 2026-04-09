package coresvc

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"

	"git.duti.dev/secure-package-registry/internal/gen/coredb"
	"git.duti.dev/secure-package-registry/internal/messages"
	"git.duti.dev/secure-package-registry/pkg/lockfile"
	"git.duti.dev/secure-package-registry/pkg/npm"
	"git.duti.dev/secure-package-registry/pkg/services/core-svc/handlers/external"
	"git.duti.dev/secure-package-registry/pkg/verification"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
)

// consumeProjectProcessing reads spr.project.processing.requested messages and
// performs async dependency resolution + storage for uploaded projects.
func consumeProjectProcessing(
	ctx context.Context,
	queries *coredb.Queries,
	publisher message.Publisher,
	npmClient *npm.Client,
	resolver *npm.Resolver,
	verifier *verification.Service,
	messagesCh <-chan *message.Message,
) error {
	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Stopping project-processing consumer")
			return nil
		case msg, ok := <-messagesCh:
			if !ok {
				log.Info().Msg("Project-processing subscription closed")
				return nil
			}
			if err := handleProjectProcessing(ctx, queries, publisher, npmClient, resolver, verifier, msg); err != nil {
				log.Error().Err(err).Msg("Failed to handle project-processing message")
				msg.Nack()
				continue
			}
			msg.Ack()
		}
	}
}

func handleProjectProcessing(
	ctx context.Context,
	queries *coredb.Queries,
	publisher message.Publisher,
	npmClient *npm.Client,
	resolver *npm.Resolver,
	verifier *verification.Service,
	msg *message.Message,
) error {
	var req messages.ProjectProcessingRequested
	if err := gob.NewDecoder(bytes.NewReader(msg.Payload)).Decode(&req); err != nil {
		return fmt.Errorf("decoding project-processing message: %w", err)
	}

	l := log.With().
		Int32("project_id", req.ProjectID).
		Int32("generation", req.Generation).
		Logger()

	// Fetch project with source file.
	project, err := queries.GetProjectForProcessing(ctx, req.ProjectID)
	if errors.Is(err, pgx.ErrNoRows) {
		l.Warn().Msg("Project not found (deleted?), discarding message")
		return nil
	}
	if err != nil {
		return fmt.Errorf("fetching project %d: %w", req.ProjectID, err)
	}

	// Stale-message guard: if the project's generation has advanced past the
	// message's generation, a newer upload superseded this one. Discard.
	if project.Generation != req.Generation {
		l.Info().
			Int32("current_generation", project.Generation).
			Msg("Stale message (generation mismatch), discarding")
		return nil
	}

	if len(project.SourceFile) == 0 {
		l.Warn().Msg("Project has no source file (already processed?), discarding")
		return nil
	}

	l.Info().Str("source_type", project.SourceType).Msg("Processing project upload")

	// Parse the stored file.
	parsed, err := lockfile.Parse(project.SourceFile)
	if err != nil {
		l.Error().Err(err).Msg("Failed to parse stored source file")
		finishErr := queries.FinishProjectProcessing(ctx, coredb.FinishProjectProcessingParams{
			ID:         req.ProjectID,
			Status:     "failed",
			Generation: req.Generation,
		})
		if finishErr != nil {
			l.Error().Err(finishErr).Msg("Failed to mark project as failed")
		}
		return nil // don't retry — the file is invalid
	}

	// --- Phase 1: Resolve direct dep constraints (package.json only) ---
	var directDeps []lockfile.Dep
	skipped := 0
	for _, dep := range parsed.DirectDeps() {
		if external.IsNonStandardSpecifier(dep.Constraint) {
			l.Warn().
				Str("dep", dep.Name).
				Str("specifier", dep.Constraint).
				Msg("Skipping non-standard dependency specifier")
			skipped++
			continue
		}

		version := dep.Version
		if version == "" && dep.Constraint != "" {
			resolved, err := npmClient.ResolveConstraint(ctx, dep.Name, dep.Constraint)
			if err != nil {
				l.Warn().Err(err).
					Str("dep", dep.Name).
					Str("constraint", dep.Constraint).
					Msg("Failed to resolve version constraint — skipping dependency")
				skipped++
				continue
			}
			version = resolved
		}
		if version == "" {
			l.Warn().Str("dep", dep.Name).Msg("No version or constraint — skipping dependency")
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
		transitiveDeps := external.ResolveTransitiveDeps(ctx, resolver, l, directDeps)
		allDeps = make([]lockfile.Dep, 0, len(directDeps)+len(transitiveDeps))
		allDeps = append(allDeps, directDeps...)
		allDeps = append(allDeps, transitiveDeps...)
	} else {
		// Lock file already has the complete dependency set.
		allDeps = parsed.All
	}

	// Atomically replace deps: delete all existing, then insert new.
	if err := queries.DeleteProjectDependencies(ctx, req.ProjectID); err != nil {
		return fmt.Errorf("clearing project dependencies: %w", err)
	}

	// --- Phase 3: Store and trigger analysis ---
	for _, dep := range allDeps {
		version := dep.Version
		if version == "" && dep.Constraint != "" {
			resolved, err := npmClient.ResolveConstraint(ctx, dep.Name, dep.Constraint)
			if err != nil {
				l.Warn().Err(err).
					Str("dep", dep.Name).
					Str("constraint", dep.Constraint).
					Msg("Failed to resolve version constraint — skipping dependency")
				continue
			}
			version = resolved
		}
		if version == "" {
			l.Warn().Str("dep", dep.Name).Msg("No version or constraint — skipping dependency")
			continue
		}

		// Upsert the package.
		pkgID, err := queries.InsertPackage(ctx, coredb.InsertPackageParams{
			Identifier:    dep.Name,
			Ecosystem:     coredb.EcosystemNpm,
			LatestVersion: pgtype.Text{},
		})
		if err != nil {
			l.Error().Err(err).Str("dep", dep.Name).Msg("Failed to upsert package")
			continue
		}

		// Upsert the package version.
		pvID, err := queries.InsertPackageVersion(ctx, coredb.InsertPackageVersionParams{
			PackageID: pkgID,
			Version:   version,
			SourceUrl: pgtype.Text{},
		})
		if err != nil {
			l.Error().Err(err).Str("dep", dep.Name+"@"+version).Msg("Failed to upsert package version")
			continue
		}

		// Determine dependency type.
		depType := coredb.DependencyTypeTransitive
		if dep.Direct {
			depType = coredb.DependencyTypeDirect
		}

		// Insert the project dependency.
		if err := queries.InsertProjectDependency(ctx, coredb.InsertProjectDependencyParams{
			ProjectID:        req.ProjectID,
			PackageID:        pkgID,
			PackageVersionID: pvID,
			DependencyType:   depType,
			VersionConstraint: pgtype.Text{
				String: dep.Constraint,
				Valid:  dep.Constraint != "",
			},
		}); err != nil {
			l.Error().Err(err).Str("dep", dep.Name+"@"+version).Msg("Failed to insert project dependency")
			continue
		}

		// Auto-verify (attestation + OSS rebuild) for ALL deps, best-effort.
		if _, verErr := verifier.CheckAndTag(ctx, "npm", dep.Name, version); verErr != nil {
			l.Debug().Err(verErr).Str("dep", dep.Name+"@"+version).Msg("Auto-verification failed (non-fatal)")
		}

		// Trigger behavioral analysis for DIRECT deps only.
		if dep.Direct {
			triggerBehavioralAnalysis(ctx, queries, publisher, l, dep.Name, version, pvID)
		}
	}

	// Mark processing as complete.
	if err := queries.FinishProjectProcessing(ctx, coredb.FinishProjectProcessingParams{
		ID:         req.ProjectID,
		Status:     "complete",
		Generation: req.Generation,
	}); err != nil {
		return fmt.Errorf("marking project %d as complete: %w", req.ProjectID, err)
	}

	l.Info().
		Int("total_deps", len(allDeps)).
		Int("skipped", skipped).
		Msg("Project processing complete")

	return nil
}

// triggerBehavioralAnalysis creates a collection task and publishes a scan
// request for a single package version.
func triggerBehavioralAnalysis(
	ctx context.Context,
	queries *coredb.Queries,
	publisher message.Publisher,
	l zerolog.Logger,
	identifier, version string,
	pvID int32,
) {
	// Dedup: skip if already active.
	active, err := queries.HasActiveCollectionTask(ctx, coredb.HasActiveCollectionTaskParams{
		PackageVersionID: pvID,
		Source:           "npm",
	})
	if err != nil {
		l.Error().Err(err).Str("dep", identifier+"@"+version).Msg("Failed to check active collection task")
		return
	}
	if active {
		return
	}

	task, err := queries.InsertCollectionTask(ctx, coredb.InsertCollectionTaskParams{
		PackageVersionID: pvID,
		Source:           "npm",
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return // conflict — task already exists
	}
	if err != nil {
		l.Error().Err(err).Str("dep", identifier+"@"+version).Msg("Failed to insert collection task")
		return
	}

	collReq := messages.CollectionRequested{
		TaskID:     task.ID,
		Ecosystem:  "npm",
		Identifier: identifier,
		Version:    version,
	}
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(collReq); err != nil {
		l.Error().Err(err).Msg("Failed to encode collection request")
		return
	}
	if err := publisher.Publish("spr.collection.requested", message.NewMessage(watermill.NewUUID(), buf.Bytes())); err != nil {
		l.Error().Err(err).Str("dep", identifier+"@"+version).Msg("Failed to publish collection request")
	}

	l.Info().Str("dep", identifier+"@"+version).Int32("task_id", task.ID).Msg("Triggered behavioral analysis")
}
