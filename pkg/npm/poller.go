package npm

import (
	"context"
	"time"

	"git.duti.dev/secure-package-registry/pkg/logger"
	"github.com/rs/zerolog"
)

// Update represents a package update from the NPM registry.
type Update struct {
	PackageName string
}

// Poller polls the NPM CouchDB changes feed and streams package updates.
type Poller struct {
	client       *Client
	pollInterval time.Duration
	batchSize    int
	log          zerolog.Logger
}

// NewPoller creates a new poller with the given HTTP timeout.
func NewPoller(client *Client) *Poller {
	return &Poller{
		client:       client,
		pollInterval: 30 * time.Second,
		batchSize:    100,
		log:          logger.WithComponent("poller"),
	}
}

// Start begins polling the NPM changes feed.
// If lastSeq is nil, starts from the current registry position.
// Returns a channel that streams Update events. The channel is closed when
// the context is cancelled or an unrecoverable error occurs.
func (p *Poller) Start(ctx context.Context, lastSeq *SequenceID) (<-chan Update, error) {
	since, err := p.resolveStartSequence(ctx, lastSeq)
	if err != nil {
		return nil, err
	}

	updates := make(chan Update, p.batchSize)

	go p.run(ctx, since, updates)

	return updates, nil
}

// resolveStartSequence determines where to start polling from.
// If lastSeq is provided and non-empty, uses it. Otherwise starts from current.
func (p *Poller) resolveStartSequence(ctx context.Context, lastSeq *SequenceID) (SequenceID, error) {
	if lastSeq != nil && lastSeq.String() != "" {
		p.log.Info().Str("sequence", lastSeq.String()).Msg("Starting from provided sequence")
		return *lastSeq, nil
	}

	info, err := p.client.GetRegistryInfo(ctx)
	if err != nil {
		return SequenceID{}, err
	}

	p.log.Info().Str("sequence", info.UpdateSequence.String()).Msg("Starting from current registry position")
	return info.UpdateSequence, nil
}

// run is the main polling loop.
func (p *Poller) run(ctx context.Context, since SequenceID, updates chan<- Update) {
	defer close(updates)

	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	pollNow := make(chan struct{}, 1)
	currentSeq := since

	p.log.Info().
		Dur("interval", p.pollInterval).
		Int("batch_size", p.batchSize).
		Msg("Starting poller")

	// Trigger initial poll
	select {
	case pollNow <- struct{}{}:
	default:
	}

	for {
		select {
		case <-ctx.Done():
			p.log.Info().Msg("Stopping poller")
			return

		case <-pollNow:
			if p.pollOnce(ctx, &currentSeq, updates) {
				// Got a full batch, trigger immediate follow-up poll
				select {
				case pollNow <- struct{}{}:
				default:
				}
			}

		case <-ticker.C:
			if p.pollOnce(ctx, &currentSeq, updates) {
				// Got a full batch, trigger immediate follow-up poll
				select {
				case pollNow <- struct{}{}:
				default:
				}
			}
		}
	}
}

// pollOnce performs a single poll cycle.
// Returns true if there may be more changes (got a full batch).
func (p *Poller) pollOnce(ctx context.Context, since *SequenceID, updates chan<- Update) bool {
	resp, err := p.client.PollChanges(ctx, *since, p.batchSize)
	if err != nil {
		p.log.Error().Err(err).Msg("Failed to poll changes")
		return false
	}

	if len(resp.Results) == 0 {
		return false
	}

	p.log.Debug().
		Int("changes", len(resp.Results)).
		Str("last_seq", resp.LastSeq.String()).
		Msg("Received changes")

	for _, change := range resp.Results {
		if change.IsDesignDoc() || change.Deleted {
			continue
		}

		updates <- Update{PackageName: change.ID}
	}

	*since = resp.LastSeq
	return len(resp.Results) >= p.batchSize
}
