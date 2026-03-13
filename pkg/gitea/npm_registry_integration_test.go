//go:build integration

package gitea

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const integrationBaseURL = "https://git.duti.dev"

func TestNpmRegistryLiveUploadAndDelete(t *testing.T) {
	t.Parallel()

	token := strings.TrimSpace(os.Getenv("TEST_GITEA_TOKEN"))
	if token == "" {
		t.Skip("TEST_GITEA_TOKEN not set")
	}

	username := strings.TrimSpace(os.Getenv("TEST_GITEA_USERNAME"))
	if username == "" {
		t.Skip("TEST_GITEA_USERNAME not set")
	}

	ctx := context.Background()
	client := NewClient(integrationBaseURL, &Config{
		Accounts: map[string]Account{
			"test": {
				Username: username,
				Token:    token,
			},
		},
	})

	registry, err := client.NpmRegistry("test")
	require.NoError(t, err)

	packageName := fmt.Sprintf("spr-integration-%d", time.Now().UnixNano())
	version := "1.0.0"
	tarball := []byte("integration tarball")
	metadata := map[string]any{
		"description": "secure-package-registry integration test package",
	}

	exists, err := registry.PackageVersionExists(ctx, packageName, version)
	require.NoError(t, err)
	require.False(t, exists)

	err = registry.UploadPackage(ctx, packageName, version, tarball, metadata)
	require.NoError(t, err)
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cleanupErr := registry.DeletePackage(cleanupCtx, packageName, version)
		assert.NoError(t, cleanupErr)
	}()

	exists, err = registry.PackageVersionExists(ctx, packageName, version)
	require.NoError(t, err)
	assert.True(t, exists)

	packageMetadata, err := registry.GetPackageMetadata(ctx, packageName)
	require.NoError(t, err)
	versions, ok := packageMetadata["versions"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, versions, version)
}
