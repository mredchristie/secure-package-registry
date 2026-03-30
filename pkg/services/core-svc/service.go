// Package coresvc implements the core API service that serves external and internal HTTP endpoints.
// It also consumes spr.package.updated messages to create collection tasks and publish
// spr.collection.requested messages, and consumes spr.collection.completed messages to
// update collection task status.
package coresvc

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"git.duti.dev/secure-package-registry/internal/gen/coredb"
	"git.duti.dev/secure-package-registry/internal/messages"
	"git.duti.dev/secure-package-registry/pkg/behavior"
	"git.duti.dev/secure-package-registry/pkg/gitea"
	"git.duti.dev/secure-package-registry/pkg/logger"
	sprminio "git.duti.dev/secure-package-registry/pkg/minio"
	"git.duti.dev/secure-package-registry/pkg/npm"
	ossrebuild "git.duti.dev/secure-package-registry/pkg/oss-rebuild"
	"git.duti.dev/secure-package-registry/pkg/pkgdb"
	"git.duti.dev/secure-package-registry/pkg/services"
	"git.duti.dev/secure-package-registry/pkg/services/core-svc/server"
	"git.duti.dev/secure-package-registry/pkg/verification"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
)

var log zerolog.Logger

func init() {
	log = logger.WithComponent("core-svc")
}

// Start runs the core-svc: HTTP servers, the spr.package.updated consumer,
// and the spr.collection.completed consumer.
// Blocks until ctx is cancelled, then gracefully shuts down.
func Start(ctx context.Context, deps *services.Deps) error {
	db := pkgdb.NewClient(deps.Pool)
	queries := coredb.New(deps.Pool)

	publisher, err := amqp.NewPublisher(
		amqp.NewDurableQueueConfig(deps.Config.RabbitMQURL),
		watermill.NewStdLogger(false, false),
	)
	if err != nil {
		return fmt.Errorf("creating AMQP publisher: %w", err)
	}
	defer func() {
		if cerr := publisher.Close(); cerr != nil {
			log.Warn().Err(cerr).Msg("Failed to close AMQP publisher")
		}
	}()

	// Subscriber for spr.package.updated.
	updatedSub, err := amqp.NewSubscriber(
		amqp.NewDurableQueueConfig(deps.Config.RabbitMQURL),
		watermill.NewStdLogger(false, false),
	)
	if err != nil {
		return fmt.Errorf("creating package-updated subscriber: %w", err)
	}
	defer func() {
		if cerr := updatedSub.Close(); cerr != nil {
			log.Warn().Err(cerr).Msg("Failed to close package-updated subscriber")
		}
	}()

	updatedCh, err := updatedSub.Subscribe(ctx, "spr.package.updated")
	if err != nil {
		return fmt.Errorf("subscribing to package updates: %w", err)
	}

	// Subscriber for spr.collection.completed (separate AMQP connection per convention).
	completedSub, err := amqp.NewSubscriber(
		amqp.NewDurableQueueConfig(deps.Config.RabbitMQURL),
		watermill.NewStdLogger(false, false),
	)
	if err != nil {
		return fmt.Errorf("creating collection-completed subscriber: %w", err)
	}
	defer func() {
		if cerr := completedSub.Close(); cerr != nil {
			log.Warn().Err(cerr).Msg("Failed to close collection-completed subscriber")
		}
	}()

	completedCh, err := completedSub.Subscribe(ctx, "spr.collection.completed")
	if err != nil {
		return fmt.Errorf("subscribing to collection completions: %w", err)
	}

	// Create MinIO client for admin artifact downloads.
	minioCfg := deps.Config.MinIO
	minioClient, err := sprminio.NewClient(ctx, sprminio.Config{
		Endpoint:  minioCfg.Endpoint,
		AccessKey: minioCfg.AccessKey,
		SecretKey: minioCfg.SecretKey,
		UseSSL:    minioCfg.UseSSL,
		Bucket:    minioCfg.Bucket,
	})
	if err != nil {
		return fmt.Errorf("creating minio client: %w", err)
	}

	// Create npm and OSS rebuild clients for verification.
	npmClient := npm.NewClient(deps.Config.NPM)
	ossClient := ossrebuild.NewClient(deps.Config.OSSRebuild)
	verifier := verification.NewService(queries, npmClient, ossClient)

	// Load the Gitea config from Valkey for the registry account.
	giteaConfig, err := deps.Valkey.GetGiteaConfig(ctx)
	if err != nil {
		return fmt.Errorf("loading gitea config from valkey: %w", err)
	}
	giteaClient := gitea.NewClient(giteaConfig)
	registryAccount, err := giteaClient.NpmRegistry("registry")
	if err != nil {
		return fmt.Errorf("creating gitea registry account client: %w", err)
	}

	// Run consumers in separate goroutines; report fatal errors via channels.
	updatedErr := make(chan error, 1)
	go func() {
		if err := consumePackageUpdated(ctx, queries, publisher, verifier, updatedCh); err != nil {
			updatedErr <- err
		}
	}()

	completedErr := make(chan error, 1)
	go func() {
		if err := consumeCollectionCompleted(ctx, queries, minioClient, completedCh); err != nil {
			completedErr <- err
		}
	}()

	externalServer := server.NewExternal("0.0.0.0:"+deps.Config.CoreSvc.ExternalPort, db, server.AdminDeps{
		Querier:   queries,
		Publisher: publisher,
		MinIO:     minioClient,
	}, verifier)
	internalServer := server.NewInternal("0.0.0.0:"+deps.Config.CoreSvc.InternalPort, server.InternalDeps{
		Querier:         queries,
		Publisher:       publisher,
		RegistryAccount: registryAccount,
	})

	externalErrCh := externalServer.Start()
	internalErrCh := internalServer.Start()

	select {
	case <-ctx.Done():
		log.Info().Msg("Shutting down servers...")
	case err := <-externalErrCh:
		return fmt.Errorf("external server: %w", err)
	case err := <-internalErrCh:
		return fmt.Errorf("internal server: %w", err)
	case err := <-updatedErr:
		return fmt.Errorf("package-updated consumer: %w", err)
	case err := <-completedErr:
		return fmt.Errorf("collection-completed consumer: %w", err)
	}

	shutdownCtx := context.Background()
	var shutdownErr error
	if err := externalServer.Stop(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Failed to stop external server")
		shutdownErr = err
	}
	if err := internalServer.Stop(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Failed to stop internal server")
		if shutdownErr == nil {
			shutdownErr = err
		}
	}

	log.Info().Msg("Shutdown complete")
	return shutdownErr
}

// consumePackageUpdated reads spr.package.updated messages and, for each one:
//  1. Looks up the package by ecosystem+identifier.
//  2. Upserts the package version (source_url left null — the poller doesn't have it).
//  3. Checks whether an active collection task already exists (dedup).
//  4. If not, inserts a pending collection task and publishes spr.collection.requested.
func consumePackageUpdated(
	ctx context.Context,
	queries *coredb.Queries,
	publisher message.Publisher,
	verifier *verification.Service,
	messagesCh <-chan *message.Message,
) error {
	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Stopping package-updated consumer")
			return nil
		case msg, ok := <-messagesCh:
			if !ok {
				log.Info().Msg("Package-updated subscription closed")
				return nil
			}
			if err := handlePackageUpdated(ctx, queries, publisher, verifier, msg); err != nil {
				log.Error().Err(err).Msg("Failed to handle package-updated message")
				msg.Nack()
				continue
			}
			msg.Ack()
		}
	}
}

func handlePackageUpdated(
	ctx context.Context,
	queries *coredb.Queries,
	publisher message.Publisher,
	verifier *verification.Service,
	msg *message.Message,
) error {
	var upd messages.PackageUpdated
	if err := gob.NewDecoder(bytes.NewReader(msg.Payload)).Decode(&upd); err != nil {
		return fmt.Errorf("decoding package-updated message: %w", err)
	}

	l := log.With().
		Str("ecosystem", upd.Ecosystem).
		Str("package", upd.Identifier).
		Str("version", upd.Version).
		Logger()

	// Look up the package. It must already exist because the poller created it.
	pkg, err := queries.GetPackageByEcosystemAndIdentifier(ctx, coredb.GetPackageByEcosystemAndIdentifierParams{
		Ecosystem:  coredb.Ecosystem(upd.Ecosystem),
		Identifier: upd.Identifier,
	})
	if err != nil {
		return fmt.Errorf("looking up package %s/%s: %w", upd.Ecosystem, upd.Identifier, err)
	}

	// Upsert the package version. source_url is null here; the COALESCE in the
	// query preserves any existing value.
	pvID, err := queries.InsertPackageVersion(ctx, coredb.InsertPackageVersionParams{
		PackageID: pkg.ID,
		Version:   upd.Version,
		SourceUrl: pgtype.Text{}, // null
	})
	if err != nil {
		return fmt.Errorf("upserting package version: %w", err)
	}

	// Update the package's latest_version to reflect the newly detected version.
	err = queries.UpdatePackageLatestVersion(ctx, coredb.UpdatePackageLatestVersionParams{
		ID:            pkg.ID,
		LatestVersion: pgtype.Text{String: upd.Version, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("updating package latest version: %w", err)
	}

	// Auto-verify: check upstream attestation and OSS rebuild status.
	// Best-effort — errors are logged but don't block ingestion.
	if _, err := verifier.CheckAndTag(ctx, upd.Ecosystem, upd.Identifier, upd.Version); err != nil {
		l.Warn().Err(err).Msg("Auto-verification failed (non-fatal)")
	} else {
		l.Info().Msg("Auto-verification completed")
	}

	// Dedup: skip if a collection task is already pending or running.
	active, err := queries.HasActiveCollectionTask(ctx, coredb.HasActiveCollectionTaskParams{
		PackageVersionID: pvID,
		Source:           "npm",
	})
	if err != nil {
		return fmt.Errorf("checking active collection task: %w", err)
	}
	if active {
		l.Debug().Msg("Active collection task already exists, skipping")
		return nil
	}

	// Insert the collection task. ON CONFLICT DO NOTHING handles races.
	task, err := queries.InsertCollectionTask(ctx, coredb.InsertCollectionTaskParams{
		PackageVersionID: pvID,
		Source:           "npm",
	})
	if errors.Is(err, pgx.ErrNoRows) {
		// Conflict — another consumer beat us to it.
		l.Debug().Msg("Collection task already exists (conflict), skipping")
		return nil
	}
	if err != nil {
		return fmt.Errorf("inserting collection task: %w", err)
	}

	// Publish spr.collection.requested for be-runner to pick up.
	collReq := messages.CollectionRequested{
		TaskID:     task.ID,
		Ecosystem:  upd.Ecosystem,
		Identifier: upd.Identifier,
		Version:    upd.Version,
	}
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(collReq); err != nil {
		return fmt.Errorf("encoding collection-requested message: %w", err)
	}
	wmMsg := message.NewMessage(watermill.NewUUID(), buf.Bytes())
	if err := publisher.Publish("spr.collection.requested", wmMsg); err != nil {
		// Log but don't fail — the task is persisted; a retry mechanism can
		// re-publish later.
		l.Error().Err(err).Msg("Failed to publish collection-requested (task persisted)")
		return nil
	}

	l.Info().Msg("Created collection task and published collection request")
	return nil
}

// consumeCollectionCompleted reads spr.collection.completed messages from be-runner
// and updates collection task status accordingly.
func consumeCollectionCompleted(
	ctx context.Context,
	queries *coredb.Queries,
	minio *sprminio.Client,
	messagesCh <-chan *message.Message,
) error {
	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Stopping collection-completed consumer")
			return nil
		case msg, ok := <-messagesCh:
			if !ok {
				log.Info().Msg("Collection-completed subscription closed")
				return nil
			}
			if err := handleCollectionCompleted(ctx, queries, minio, msg); err != nil {
				log.Error().Err(err).Msg("Failed to handle collection-completed message")
				msg.Nack()
				continue
			}
			msg.Ack()
		}
	}
}

func handleCollectionCompleted(
	ctx context.Context,
	queries *coredb.Queries,
	minio *sprminio.Client,
	msg *message.Message,
) error {
	var completed messages.CollectionCompleted
	if err := gob.NewDecoder(bytes.NewReader(msg.Payload)).Decode(&completed); err != nil {
		return fmt.Errorf("decoding collection-completed message: %w", err)
	}

	l := log.With().
		Int32("task_id", completed.TaskID).
		Str("ecosystem", completed.Ecosystem).
		Str("package", completed.Identifier).
		Str("version", completed.Version).
		Logger()

	if completed.Success {
		err := queries.UpdateCollectionTaskSucceeded(ctx, coredb.UpdateCollectionTaskSucceededParams{
			ID:             completed.TaskID,
			ArtifactBucket: pgtype.Text{String: completed.ArtifactBucket, Valid: true},
			ArtifactKey:    pgtype.Text{String: completed.ArtifactKey, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("marking task %d succeeded: %w", completed.TaskID, err)
		}
		l.Info().
			Str("bucket", completed.ArtifactBucket).
			Str("key", completed.ArtifactKey).
			Msg("Collection task succeeded")

		// Best-effort: evaluate the deduped behavior tree and set the
		// behavior_passed tag. An empty deduped tree (no anomalous behaviors
		// after baseline subtraction) means the package passed.
		if err := evaluateBehavior(ctx, queries, minio, completed); err != nil {
			l.Warn().Err(err).Msg("Failed to evaluate behavioral analysis result")
		}
	} else {
		err := queries.UpdateCollectionTaskFailed(ctx, coredb.UpdateCollectionTaskFailedParams{
			ID:            completed.TaskID,
			FailureReason: pgtype.Text{String: completed.FailureReason, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("marking task %d failed: %w", completed.TaskID, err)
		}
		l.Warn().
			Str("reason", completed.FailureReason).
			Msg("Collection task failed")
	}

	return nil
}

// evaluateBehavior downloads the deduped behavior tree from MinIO and sets the
// behavior_passed tag on the package version. An empty tree (no anomalous
// behaviors remaining after baseline subtraction) is treated as passed.
func evaluateBehavior(
	ctx context.Context,
	queries *coredb.Queries,
	minio *sprminio.Client,
	completed messages.CollectionCompleted,
) error {
	// Derive the deduped JSON key from the raw artifact key.
	// Raw:     behavior/{eco}/{pkg}/{ver}/{src}/behavior.jsonl
	// Deduped: behavior/{eco}/{pkg}/{ver}/{src}/behavior-deduped.json
	dedupedKey := strings.TrimSuffix(completed.ArtifactKey, "behavior.jsonl") + "behavior-deduped.json"

	data, err := minio.GetObject(ctx, dedupedKey)
	if err != nil {
		return fmt.Errorf("downloading deduped tree %q: %w", dedupedKey, err)
	}

	var tree behavior.ProcessTree
	if err := json.Unmarshal(data, &tree); err != nil {
		return fmt.Errorf("unmarshalling deduped tree: %w", err)
	}

	passed := tree.IsEmpty()

	// Look up the package version ID.
	pvID, err := queries.GetPackageVersionID(ctx, coredb.GetPackageVersionIDParams{
		Ecosystem:  coredb.Ecosystem(completed.Ecosystem),
		Identifier: completed.Identifier,
		Version:    completed.Version,
	})
	if err != nil {
		return fmt.Errorf("looking up package version: %w", err)
	}

	// Look up the tag type and upsert.
	tagTypeID, err := queries.GetTagTypeByLabel(ctx, "behavior_passed")
	if err != nil {
		return fmt.Errorf("looking up behavior_passed tag type: %w", err)
	}

	val := []byte(`false`)
	if passed {
		val = []byte(`true`)
	}

	if err := queries.InsertPackageTag(ctx, coredb.InsertPackageTagParams{
		PackageVersion: pvID,
		TagType:        tagTypeID,
		Value:          val,
	}); err != nil {
		return fmt.Errorf("upserting behavior_passed tag: %w", err)
	}

	log.Info().
		Str("ecosystem", completed.Ecosystem).
		Str("package", completed.Identifier).
		Str("version", completed.Version).
		Bool("passed", passed).
		Msg("Behavioral analysis evaluated")

	return nil
}
