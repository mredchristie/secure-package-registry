package packagewatcher

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"sync"

	"git.duti.dev/secure-package-registry/internal/messages"
	"git.duti.dev/secure-package-registry/pkg/logger"
	"git.duti.dev/secure-package-registry/pkg/npm"
	"git.duti.dev/secure-package-registry/pkg/services"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
)

var log = logger.WithComponent("package-watcher")

type watcher struct {
	mu sync.RWMutex
	// Ecosystem -> Package Name -> Version
	watchedPackages map[string]map[string]string
	subscriber      *amqp.Subscriber
	publisher       *amqp.Publisher
	npmClient       *npm.Client
	poller          *npm.Poller
}

// Start creates a new package watcher and runs it. Blocks until ctx is cancelled.
func Start(ctx context.Context, deps *services.Deps) error {
	w, err := NewWatcher(deps)
	if err != nil {
		return err
	}
	return w.Start(ctx)
}

// NewWatcher creates a new package watcher using shared dependencies.
// It creates its own AMQP subscriber/publisher from deps.Config.RabbitMQURL.
func NewWatcher(deps *services.Deps) (*watcher, error) {
	subscriber, err := amqp.NewSubscriber(
		amqp.NewDurableQueueConfig(deps.Config.RabbitMQURL),
		watermill.NewStdLogger(false, false),
	)
	if err != nil {
		return nil, fmt.Errorf("creating AMQP subscriber: %w", err)
	}
	publisher, err := amqp.NewPublisher(
		amqp.NewDurableQueueConfig(deps.Config.RabbitMQURL),
		watermill.NewStdLogger(false, false),
	)
	if err != nil {
		return nil, fmt.Errorf("creating AMQP publisher: %w", err)
	}
	npmClient := npm.NewClient(deps.Config.NPM)
	return &watcher{
		watchedPackages: make(map[string]map[string]string),
		subscriber:      subscriber,
		publisher:       publisher,
		npmClient:       npmClient,
		poller:          npm.NewPoller(npmClient),
	}, nil
}

func (w *watcher) Start(ctx context.Context) error {
	requestsCh, err := w.subscriber.Subscribe(context.Background(), "spr.package.requested")
	if err != nil {
		return fmt.Errorf("failed to subscribe to package requests: %w", err)
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-requestsCh:
				var req messages.PackageRequest
				if err := gob.NewDecoder(bytes.NewReader(msg.Payload)).Decode(&req); err != nil {
					// A payload that fails to decode will never succeed on
					// redelivery; drop it instead of requeueing forever.
					log.Error().Err(err).Msg("Dropping undecodable package request")
					msg.Ack()
					continue
				}
				log.Info().
					Str("ecosystem", req.Ecosystem).
					Str("identifier", req.Identifier).
					Msg("Received package request")
				// Handle the package request by triggering fetching the initial version and adding to watch list
				w.insertWatchedPackage(req.Ecosystem, req.Identifier, "")
				if err := w.checkVersionAndPublish(ctx, req.Ecosystem, req.Identifier); err != nil {
					// Ack rather than Nack: requeueing a package that does not
					// exist upstream (e.g. a typo) redelivers it in a hot loop.
					// Transient upstream errors are retried by the periodic
					// version poll since the package is on the watch list.
					log.Error().
						Err(err).
						Str("identifier", req.Identifier).
						Msg("Failed to check version and publish update, dropping request")
					msg.Ack()
					continue
				}
				msg.Ack()
			}
		}
	}()
	repErrCh := make(chan error, 1)
	go func() {
		if repErr := w.listenNPMReplicate(ctx); repErr != nil {
			log.Error().Err(repErr).Msg("NPM replicate listener stopped with error")
			repErrCh <- fmt.Errorf("NPM replicate listener error: %w", repErr)
		} else {
			log.Info().Msg("NPM replicate listener stopped gracefully")
			repErrCh <- nil
		}
	}()
	select {
	case <-ctx.Done():
		return nil
	case err := <-repErrCh:
		return err
	}
}

func (w *watcher) insertWatchedPackage(ecosystem, identifier, version string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, exists := w.watchedPackages[ecosystem]; !exists {
		w.watchedPackages[ecosystem] = make(map[string]string)
	}
	w.watchedPackages[ecosystem][identifier] = version
}

func (w *watcher) checkVersionAndPublish(ctx context.Context, ecosystem, identifier string) error {
	if ecosystem != "npm" {
		log.Warn().
			Str("ecosystem", ecosystem).
			Str("identifier", identifier).
			Msg("Unsupported ecosystem, skipping version check")
		return fmt.Errorf("unsupported ecosystem: %s", ecosystem)
	}
	latestVersion, err := w.npmClient.GetLatestVersion(ctx, identifier)
	if err != nil {
		return fmt.Errorf("failed to fetch latest version for %s: %w", identifier, err)
	}
	currentVersion, _ := w.currentWatchedVersion(ecosystem, identifier)
	if latestVersion == currentVersion {
		return nil
	}
	log.Info().
		Str("identifier", identifier).
		Str("old_version", currentVersion).
		Str("new_version", latestVersion).
		Msg("New version detected, publishing update")
	w.insertWatchedPackage(ecosystem, identifier, latestVersion)
	updateMsg := messages.PackageUpdated{
		Ecosystem:  ecosystem,
		Identifier: identifier,
		Version:    latestVersion,
	}
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(updateMsg); err != nil {
		return fmt.Errorf("failed to encode package update message: %w", err)
	}
	return w.publisher.Publish("spr.package.updated", message.NewMessage(watermill.NewUUID(), buf.Bytes()))
}

func (w *watcher) currentWatchedVersion(ecosystem, identifier string) (string, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if pkgs, exists := w.watchedPackages[ecosystem]; exists {
		version, exists := pkgs[identifier]
		return version, exists
	}
	return "", false
}

func (w *watcher) listenNPMReplicate(ctx context.Context) error {
	updatesCh, err := w.poller.Start(ctx, nil) // TODO: persist last sequence ID to avoid missing updates on restart
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case update := <-updatesCh:
			// First ensure that we're actively watching this package before publishing an update
			_, watching := w.currentWatchedVersion("npm", update.PackageName)
			if !watching {
				continue
			}
			// Fetch the latest version
			if err := w.checkVersionAndPublish(ctx, "npm", update.PackageName); err != nil {
				log.Error().
					Err(err).
					Str("package", update.PackageName).
					Msg("Failed to check version and publish update")
				continue
			}

		}
	}
}
