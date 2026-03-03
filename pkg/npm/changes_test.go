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

func newTestClient(t *testing.T) *npm.Client {
	rec := vcr.NewRecorder(t)
	return npm.NewClient(config.NPMConfig{
		HTTPTimeout:   10 * time.Second,
		HTTPTransport: rec,
	})
}

func TestGetLatestVersion(t *testing.T) {
	t.Parallel()

	t.Run("returns latest version for express", func(t *testing.T) {
		t.Parallel()
		client := newTestClient(t)
		version, err := client.GetLatestVersion(context.Background(), "express")

		require.NoError(t, err)
		assert.NotEmpty(t, version)
	})

	t.Run("errors on nonexistent package", func(t *testing.T) {
		t.Parallel()
		client := newTestClient(t)
		_, err := client.GetLatestVersion(context.Background(), "this-package-definitely-does-not-exist-abc123xyz")

		assert.Error(t, err)
	})
}

func TestPollChanges(t *testing.T) {
	t.Parallel()

	t.Run("fetches changes with limit", func(t *testing.T) {
		t.Parallel()
		client := newTestClient(t)
		resp, err := client.PollChanges(context.Background(), npm.NewSequenceID("99543500"), 5)

		require.NoError(t, err)
		assert.Len(t, resp.Results, 5)
		assert.NotEmpty(t, resp.LastSeq.String())

		for _, change := range resp.Results {
			assert.NotEmpty(t, change.ID)
			assert.NotEmpty(t, change.Sequence.String())
		}
	})
}

func TestGetRegistryInfo(t *testing.T) {
	t.Parallel()

	client := newTestClient(t)
	info, err := client.GetRegistryInfo(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "registry", info.DBName)
	assert.Greater(t, info.DocCount, int64(0))
	assert.NotEmpty(t, info.UpdateSequence.String())
}
