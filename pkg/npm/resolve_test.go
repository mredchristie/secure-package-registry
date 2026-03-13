package npm_test

import (
	"context"
	"testing"

	"git.duti.dev/secure-package-registry/pkg/npm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolve(t *testing.T) {
	t.Parallel()

	t.Run("resolves has-flag with zero dependencies", func(t *testing.T) {
		t.Parallel()
		client := newTestClient(t)
		resolver := npm.NewResolver(client)

		graph, err := resolver.Resolve(context.Background(), "has-flag", "4.0.0")
		require.NoError(t, err)

		assert.Equal(t, "has-flag@4.0.0", graph.Root)
		require.Contains(t, graph.Nodes, "has-flag@4.0.0")
		assert.Equal(t, "has-flag", graph.Nodes["has-flag@4.0.0"].Name)
		assert.Equal(t, "4.0.0", graph.Nodes["has-flag@4.0.0"].Version)
		assert.NotEmpty(t, graph.Nodes["has-flag@4.0.0"].TarballURL)
		assert.Len(t, graph.Nodes, 1)
		assert.Empty(t, graph.Edges["has-flag@4.0.0"])
	})

	t.Run("resolves supports-color with transitive dependencies", func(t *testing.T) {
		t.Parallel()
		client := newTestClient(t)
		resolver := npm.NewResolver(client)

		graph, err := resolver.Resolve(context.Background(), "supports-color", "7.2.0")
		require.NoError(t, err)

		assert.Equal(t, "supports-color@7.2.0", graph.Root)
		require.Contains(t, graph.Nodes, "supports-color@7.2.0")
		require.Contains(t, graph.Nodes, "has-flag@4.0.0")

		rootEdges := graph.Edges["supports-color@7.2.0"]
		assert.Contains(t, rootEdges, "has-flag@4.0.0")

		for key, node := range graph.Nodes {
			assert.NotNil(t, node, "node %s should not be nil", key)
			assert.NotEmpty(t, node.TarballURL, "node %s should have tarball URL", key)
		}
	})

	t.Run("deduplicates shared dependencies", func(t *testing.T) {
		t.Parallel()
		client := newTestClient(t)
		resolver := npm.NewResolver(client)

		graph, err := resolver.Resolve(context.Background(), "chalk", "4.1.2")
		require.NoError(t, err)

		seen := make(map[string]bool)
		for key := range graph.Nodes {
			assert.False(t, seen[key], "duplicate node: %s", key)
			seen[key] = true
		}

		require.Contains(t, graph.Nodes, "chalk@4.1.2")
		require.Contains(t, graph.Nodes, "supports-color@7.2.0")
		require.Contains(t, graph.Nodes, "ansi-styles@4.3.0")
	})

	t.Run("rejects invalid semver version", func(t *testing.T) {
		t.Parallel()
		client := newTestClient(t)
		resolver := npm.NewResolver(client)

		_, err := resolver.Resolve(context.Background(), "has-flag", "not-a-version")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not valid semver")
	})

	t.Run("errors on nonexistent package", func(t *testing.T) {
		t.Parallel()
		client := newTestClient(t)
		resolver := npm.NewResolver(client)

		_, err := resolver.Resolve(context.Background(), "this-package-absolutely-does-not-exist-xyz", "1.0.0")
		assert.Error(t, err)
	})
}

func TestMatchVersion(t *testing.T) {
	t.Parallel()

	t.Run("resolves caret range", func(t *testing.T) {
		t.Parallel()
		client := newTestClient(t)
		resolver := npm.NewResolver(client)

		graph, err := resolver.Resolve(context.Background(), "supports-color", "7.2.0")
		require.NoError(t, err)

		require.Contains(t, graph.Nodes, "has-flag@4.0.0")
	})
}
