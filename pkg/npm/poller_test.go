package npm_test

import (
	"context"
	"testing"
	"time"

	"charm.land/x/vcr"
	"git.duti.dev/secure-package-registry/pkg/config"
	"git.duti.dev/secure-package-registry/pkg/npm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPollerStreamsUpdates(t *testing.T) {
	t.Parallel()

	rec := vcr.NewRecorder(t)
	client := npm.NewClient(config.NPMConfig{
		HTTPTimeout:   10 * time.Second,
		HTTPTransport: rec,
	})
	poller := npm.NewPoller(client)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	seq := npm.NewSequenceID("99543500")
	updates, err := poller.Start(ctx, &seq)
	require.NoError(t, err)

	var received []string
	for i := 0; i < 5; i++ {
		select {
		case u := <-updates:
			received = append(received, u.PackageName)
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for update")
		}
	}
	cancel()

	assert.Len(t, received, 5)
	for _, name := range received {
		assert.NotEmpty(t, name)
		assert.NotContains(t, name, "_design")
	}
}

func TestPollerStartsFromCurrentWhenNoSequence(t *testing.T) {
	t.Parallel()

	rec := vcr.NewRecorder(t)
	client := npm.NewClient(config.NPMConfig{
		HTTPTimeout:   10 * time.Second,
		HTTPTransport: rec,
	})
	poller := npm.NewPoller(client)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	updates, err := poller.Start(ctx, nil)
	require.NoError(t, err)

	<-ctx.Done()
	for range updates {
	}
}
