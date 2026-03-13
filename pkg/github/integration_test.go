//go:build integration

package github

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowDispatchAndArtifactDownload(t *testing.T) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		t.Skip("GITHUB_TOKEN not set")
	}
	owner := os.Getenv("GITHUB_OWNER")
	if owner == "" {
		t.Skip("GITHUB_OWNER not set")
	}
	repo := os.Getenv("GITHUB_REPO")
	if repo == "" {
		t.Skip("GITHUB_REPO not set")
	}

	c := NewClient(Config{
		Token: token,
		Owner: owner,
		Repo:  repo,
	})
	ctx := context.Background()

	// Use a tiny package from the public npm registry so the workflow finishes fast.
	inputs := map[string]string{
		"package":  "is-odd",
		"version":  "3.0.1",
		"registry": "npm",
	}

	// --- Trigger ---
	t.Log("dispatching workflow")
	run, err := c.TriggerWorkflow(ctx, "collect-behavior.yml", inputs)
	require.NoError(t, err)
	require.NotZero(t, run.RunID)
	t.Logf("workflow run created: id=%d url=%s", run.RunID, run.HTMLURL)

	// --- Poll to completion ---
	// GitHub Actions runners can take a while to pick up jobs.
	pollCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	t.Log("polling for completion")
	completed, err := c.PollWorkflowRun(pollCtx, run.RunID, 15*time.Second)
	require.NoError(t, err)
	t.Logf("workflow completed: status=%s conclusion=%s", completed.Status, completed.Conclusion)
	assert.Equal(t, "completed", completed.Status)
	// The workflow may succeed or fail (e.g. if Tracee has issues on the runner),
	// but we still expect artifacts to be uploaded because of fail-open design.

	// --- List artifacts ---
	artifacts, err := c.ListArtifacts(ctx, run.RunID)
	require.NoError(t, err)
	t.Logf("found %d artifacts", len(artifacts))

	// Find the behavior artifact. Name format: behavior-{normalized}-{version}-{run_id}
	var behaviorArtifact *Artifact
	for i := range artifacts {
		if strings.HasPrefix(artifacts[i].Name, "behavior-") {
			behaviorArtifact = &artifacts[i]
			break
		}
	}
	require.NotNil(t, behaviorArtifact, "expected a behavior-* artifact")
	t.Logf("downloading artifact: id=%d name=%s size=%d", behaviorArtifact.ID, behaviorArtifact.Name, behaviorArtifact.SizeInBytes)

	// --- Download artifact ---
	data, err := c.DownloadArtifact(ctx, behaviorArtifact.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
	t.Logf("downloaded %d bytes", len(data))

	// The artifact is a zip file; verify the magic bytes.
	assert.Equal(t, []byte("PK"), data[:2], "expected zip file magic bytes")
}
