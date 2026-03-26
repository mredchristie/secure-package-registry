// Package berunner implements the behavioral analysis runner service.
// It subscribes to collection requests, mirrors packages to Gitea,
// triggers GitHub Actions workflows for behavioral analysis, stores
// resulting artifacts in MinIO, and publishes completion events.
package berunner

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"git.duti.dev/secure-package-registry/internal/messages"
	"git.duti.dev/secure-package-registry/pkg/behavior"
	"git.duti.dev/secure-package-registry/pkg/gitea"
	"git.duti.dev/secure-package-registry/pkg/github"
	"git.duti.dev/secure-package-registry/pkg/logger"
	sprminio "git.duti.dev/secure-package-registry/pkg/minio"
	"git.duti.dev/secure-package-registry/pkg/npm"
	"git.duti.dev/secure-package-registry/pkg/services"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	watermillmsg "github.com/ThreeDotsLabs/watermill/message"
	"github.com/rs/zerolog"
)

var log zerolog.Logger

func init() {
	log = logger.WithComponent("be-runner")
}

// Start runs the be-runner service: initializes clients, subscribes to
// spr.collection.requested, and processes each request through the full
// pipeline (resolve -> mirror -> trigger -> poll -> download -> store).
// On completion (success or failure), publishes spr.collection.completed.
// Blocks until ctx is cancelled.
func Start(ctx context.Context, deps *services.Deps) error {
	giteaConfig, err := deps.Valkey.GetGiteaConfig(ctx)
	if err != nil {
		return fmt.Errorf("loading Gitea config from Valkey: %w", err)
	}

	giteaClient := gitea.NewClient(giteaConfig)
	sandboxRegistry, err := giteaClient.NpmRegistry("sandbox")
	if err != nil {
		return fmt.Errorf("initializing sandbox registry: %w", err)
	}

	gh := github.NewClient(github.Config{
		Token: deps.Config.GitHub.Token,
		Owner: deps.Config.GitHub.Owner,
		Repo:  deps.Config.GitHub.Repo,
	})

	npmClient := npm.NewClient(deps.Config.NPM)
	resolver := npm.NewResolver(npmClient)
	mirrorer := gitea.NewMirrorer(sandboxRegistry, npmClient, gitea.MirrorOptions{})

	mc, err := sprminio.NewClient(ctx, sprminio.Config{
		Endpoint:  deps.Config.MinIO.Endpoint,
		AccessKey: deps.Config.MinIO.AccessKey,
		SecretKey: deps.Config.MinIO.SecretKey,
		UseSSL:    deps.Config.MinIO.UseSSL,
		Bucket:    deps.Config.MinIO.Bucket,
	})
	if err != nil {
		return fmt.Errorf("creating MinIO client: %w", err)
	}
	log.Info().Msg("Connected to MinIO")

	subscriber, err := amqp.NewSubscriber(
		amqp.NewDurableQueueConfig(deps.Config.RabbitMQURL),
		watermill.NewStdLogger(false, false),
	)
	if err != nil {
		return fmt.Errorf("creating AMQP subscriber: %w", err)
	}
	defer func() {
		if closeErr := subscriber.Close(); closeErr != nil {
			log.Warn().Err(closeErr).Msg("Failed to close AMQP subscriber")
		}
	}()

	publisher, err := amqp.NewPublisher(
		amqp.NewDurableQueueConfig(deps.Config.RabbitMQURL),
		watermill.NewStdLogger(false, false),
	)
	if err != nil {
		return fmt.Errorf("creating AMQP publisher: %w", err)
	}
	defer func() {
		if closeErr := publisher.Close(); closeErr != nil {
			log.Warn().Err(closeErr).Msg("Failed to close AMQP publisher")
		}
	}()

	messagesCh, err := subscriber.Subscribe(ctx, "spr.collection.requested")
	if err != nil {
		return fmt.Errorf("subscribing to collection requests: %w", err)
	}

	workflowFile := deps.Config.GitHub.WorkflowFile

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Shutting down be-runner")
			return nil
		case msg, ok := <-messagesCh:
			if !ok {
				log.Info().Msg("Collection subscription closed")
				return nil
			}

			var req messages.CollectionRequested
			if err := gob.NewDecoder(bytes.NewReader(msg.Payload)).Decode(&req); err != nil {
				log.Error().Err(err).Msg("Failed to decode collection request")
				msg.Nack()
				continue
			}

			if req.Ecosystem != "npm" {
				log.Warn().Str("ecosystem", req.Ecosystem).Msg("Unsupported ecosystem")
				msg.Ack()
				continue
			}

			result, err := HandleCollection(ctx, resolver, mirrorer, gh, mc, workflowFile, req)
			if err != nil {
				log.Error().
					Err(err).
					Str("package", req.Identifier).
					Str("version", req.Version).
					Msg("Collection request failed")

				publishCompletion(publisher, messages.CollectionCompleted{
					TaskID:        req.TaskID,
					Ecosystem:     req.Ecosystem,
					Identifier:    req.Identifier,
					Version:       req.Version,
					FailureReason: err.Error(),
				})
				// Ack (not Nack) after publishing the failure completion event.
				// Nacking would requeue the message, causing duplicate failure
				// events or contradictory failure-then-success on retry.
				msg.Ack()
				continue
			}

			publishCompletion(publisher, messages.CollectionCompleted{
				TaskID:         req.TaskID,
				Ecosystem:      req.Ecosystem,
				Identifier:     req.Identifier,
				Version:        req.Version,
				Success:        true,
				ArtifactBucket: result.Bucket,
				ArtifactKey:    result.Key,
			})
			msg.Ack()
		}
	}
}

// CollectionResult holds the MinIO location of a stored artifact.
type CollectionResult struct {
	Bucket string
	Key    string
}

// HandleCollection processes a single collection request through the full pipeline:
// resolve dependencies, mirror to Gitea, trigger GitHub workflow, poll to completion,
// download the behavior artifact, extract the .jsonl, and store it in MinIO.
// On success it returns the artifact location.
func HandleCollection(
	ctx context.Context,
	resolver *npm.Resolver,
	mirrorer *gitea.Mirrorer,
	gh *github.Client,
	mc *sprminio.Client,
	workflowFile string,
	req messages.CollectionRequested,
) (*CollectionResult, error) {
	graph, err := resolver.Resolve(ctx, req.Identifier, req.Version)
	if err != nil {
		return nil, fmt.Errorf("resolving dependencies: %w", err)
	}

	mirrorResult, err := mirrorer.Mirror(ctx, graph)
	if err != nil {
		return nil, fmt.Errorf("mirroring sandbox dependencies: %w", err)
	}

	log.Info().
		Str("package", req.Identifier).
		Str("version", req.Version).
		Int("uploaded", len(mirrorResult.Uploaded)).
		Int("skipped", len(mirrorResult.Skipped)).
		Msg("Mirrored package graph to sandbox")

	// Trigger behavioral analysis workflow on GitHub Actions.
	// The workflow pulls packages from the remote Gitea sandbox registry
	// where they were just mirrored.
	created, err := gh.TriggerWorkflow(ctx, workflowFile, map[string]string{
		"package":  req.Identifier,
		"version":  req.Version,
		"registry": "npm",
	})
	if err != nil {
		return nil, fmt.Errorf("triggering workflow: %w", err)
	}
	log.Info().
		Int64("run_id", created.RunID).
		Str("run_url", created.HTMLURL).
		Msg("Triggered behavioral analysis workflow")

	// Poll until the workflow run completes. The caller can shrink this deadline
	// via the parent context; 10 minutes is the default upper bound.
	pollCtx, pollCancel := context.WithTimeout(ctx, 10*time.Minute)
	defer pollCancel()

	run, err := gh.PollWorkflowRun(pollCtx, created.RunID, 15*time.Second)
	if err != nil {
		return nil, fmt.Errorf("polling workflow run %d: %w", created.RunID, err)
	}

	if run.Conclusion != "success" {
		return nil, fmt.Errorf("workflow run %d concluded with %q", run.ID, run.Conclusion)
	}

	// Retrieve the behavior artifact from the completed run.
	artifacts, err := gh.ListArtifacts(ctx, created.RunID)
	if err != nil {
		return nil, fmt.Errorf("listing artifacts for run %d: %w", created.RunID, err)
	}

	var behaviorArtifact *github.Artifact
	for _, a := range artifacts {
		if strings.HasPrefix(a.Name, "behavior-") {
			behaviorArtifact = &a
			break
		}
	}
	if behaviorArtifact == nil {
		return nil, fmt.Errorf("no behavior artifact found in run %d (%d artifacts listed)", created.RunID, len(artifacts))
	}

	data, err := gh.DownloadArtifact(ctx, behaviorArtifact.ID)
	if err != nil {
		return nil, fmt.Errorf("downloading artifact %d: %w", behaviorArtifact.ID, err)
	}

	// Extract the .jsonl file from the zip archive.
	jsonlData, err := extractJSONL(data)
	if err != nil {
		return nil, fmt.Errorf("extracting jsonl from artifact: %w", err)
	}

	// Store in MinIO using the RFC bucket key structure:
	// behavior/{ecosystem}/{package}/{version}/{source}/behavior.jsonl
	source := "npm"
	key := fmt.Sprintf("behavior/%s/%s/%s/%s/behavior.jsonl",
		req.Ecosystem, req.Identifier, req.Version, source)

	if err := mc.PutObject(ctx, key, jsonlData, "application/x-ndjson"); err != nil {
		return nil, fmt.Errorf("storing artifact in MinIO: %w", err)
	}

	log.Info().
		Str("package", req.Identifier).
		Str("version", req.Version).
		Str("bucket", mc.Bucket()).
		Str("key", key).
		Int("size_bytes", len(jsonlData)).
		Msg("Stored behavior artifact in MinIO")

	// Build process tree from the raw data and store as pre-computed JSON
	// for efficient serving to the dashboard. Two versions are stored:
	// 1. Raw tree (behavior-raw.json) — full tree with all behaviors.
	// 2. Deduped tree (behavior-deduped.json) — baseline noise removed.
	rawTreeKey := fmt.Sprintf("behavior/%s/%s/%s/%s/behavior-raw.json",
		req.Ecosystem, req.Identifier, req.Version, source)
	dedupedKey := fmt.Sprintf("behavior/%s/%s/%s/%s/behavior-deduped.json",
		req.Ecosystem, req.Identifier, req.Version, source)

	if err := buildAndStoreTrees(ctx, mc, jsonlData, rawTreeKey, dedupedKey); err != nil {
		// Tree-building failure is non-fatal — log and continue.
		// The raw artifact is already stored.
		log.Warn().Err(err).Msg("Failed to build/store behavior trees (non-fatal)")
	}

	return &CollectionResult{Bucket: mc.Bucket(), Key: key}, nil
}

// BaselineKey is the well-known MinIO key for the global safe baseline.
// This baseline is used to deduplicate behavioral noise from package scans.
// It must be pre-seeded into the MinIO bucket before scans produce useful
// deduped results (without a baseline, the full tree is stored as-is).
const BaselineKey = "behavior/baseline/behavior.jsonl"

// buildAndStoreTrees builds a process tree from raw JSONL data, stores it
// as the raw tree, then optionally deduplicates against the global baseline
// and stores the deduped tree. Both are JSON in MinIO.
func buildAndStoreTrees(ctx context.Context, mc *sprminio.Client, rawJSONL []byte, rawKey, dedupedKey string) error {
	targetTree, err := behavior.BuildTree(bytes.NewReader(rawJSONL), nil)
	if err != nil {
		return fmt.Errorf("building target tree: %w", err)
	}

	// Store the raw (non-deduped) tree.
	rawJSON, err := json.Marshal(targetTree)
	if err != nil {
		return fmt.Errorf("marshaling raw tree: %w", err)
	}
	if err := mc.PutObject(ctx, rawKey, rawJSON, "application/json"); err != nil {
		return fmt.Errorf("storing raw tree: %w", err)
	}
	log.Info().
		Str("key", rawKey).
		Int("size_bytes", len(rawJSON)).
		Msg("Stored raw behavior tree in MinIO")

	// Deduplicate against the global baseline.
	resultTree := targetTree

	// Try to load the global baseline. If it doesn't exist, we store
	// the full (non-deduped) tree — still useful for visualization.
	baselineData, err := mc.GetObject(ctx, BaselineKey)
	if err != nil {
		log.Warn().Msg("No baseline found in MinIO — storing full (non-deduped) tree")
	} else {
		baselineTree, err := behavior.BuildTree(bytes.NewReader(baselineData), nil)
		if err != nil {
			return fmt.Errorf("building baseline tree: %w", err)
		}
		resultTree = behavior.Dedupe(targetTree, baselineTree)
		log.Info().Msg("Deduped behavior tree against baseline")
	}

	treeJSON, err := json.Marshal(resultTree)
	if err != nil {
		return fmt.Errorf("marshaling deduped tree: %w", err)
	}

	if err := mc.PutObject(ctx, dedupedKey, treeJSON, "application/json"); err != nil {
		return fmt.Errorf("storing deduped tree: %w", err)
	}

	log.Info().
		Str("key", dedupedKey).
		Int("size_bytes", len(treeJSON)).
		Msg("Stored deduped behavior tree in MinIO")

	return nil
}

// extractJSONL reads a zip archive and returns the contents of the first
// .jsonl file found inside it.
func extractJSONL(zipData []byte) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("opening zip: %w", err)
	}

	for _, f := range r.File {
		if strings.HasSuffix(f.Name, ".jsonl") {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("opening %s in zip: %w", f.Name, err)
			}
			defer func() { _ = rc.Close() }()

			data, err := io.ReadAll(rc)
			if err != nil {
				return nil, fmt.Errorf("reading %s from zip: %w", f.Name, err)
			}
			return data, nil
		}
	}

	return nil, fmt.Errorf("no .jsonl file found in zip (%d entries)", len(r.File))
}

// publishCompletion encodes and publishes a CollectionCompleted message.
// Errors are logged but not returned since this is best-effort.
func publishCompletion(publisher watermillmsg.Publisher, completed messages.CollectionCompleted) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(completed); err != nil {
		log.Error().Err(err).Int32("task_id", completed.TaskID).Msg("Failed to encode completion event")
		return
	}

	msg := watermillmsg.NewMessage(watermill.NewUUID(), buf.Bytes())
	if err := publisher.Publish("spr.collection.completed", msg); err != nil {
		log.Error().Err(err).Int32("task_id", completed.TaskID).Msg("Failed to publish completion event")
	}
}
