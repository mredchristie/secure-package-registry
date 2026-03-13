package github

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ListArtifacts returns all artifacts for a workflow run.
func (c *Client) ListArtifacts(ctx context.Context, runID int64) ([]Artifact, error) {
	path := fmt.Sprintf("actions/runs/%d/artifacts", runID)
	var list artifactList
	if err := c.doJSON(ctx, "GET", path, nil, &list); err != nil {
		return nil, fmt.Errorf("listing artifacts: %w", err)
	}
	return list.Artifacts, nil
}

// DownloadArtifact downloads a workflow artifact as raw zip bytes.
//
// The GitHub API responds with a 302 redirect to a pre-signed blob storage URL.
// The redirect must be followed without the Authorization header, because the
// storage backend rejects requests that carry GitHub auth credentials.
func (c *Client) DownloadArtifact(ctx context.Context, artifactID int64) (data []byte, err error) {
	path := fmt.Sprintf("actions/artifacts/%d/zip", artifactID)

	// Use a one-off client that stops on redirect so we can strip auth headers.
	// It shares the same transport as the main client for testability.
	noRedirect := &http.Client{
		Timeout:   30 * time.Second,
		Transport: c.httpClient.Transport,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	url := fmt.Sprintf("%s/repos/%s/%s/%s", baseURL, c.owner, c.repo, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("X-GitHub-Api-Version", apiVersion)

	resp, err := noRedirect.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requesting artifact: %w", err)
	}

	if resp.StatusCode == http.StatusFound {
		location := resp.Header.Get("Location")
		_ = resp.Body.Close()
		if location == "" {
			return nil, fmt.Errorf("redirect with no Location header")
		}
		// Follow the redirect without auth headers.
		redirectReq, err := http.NewRequestWithContext(ctx, http.MethodGet, location, nil)
		if err != nil {
			return nil, fmt.Errorf("creating redirect request: %w", err)
		}
		resp, err = c.httpClient.Do(redirectReq)
		if err != nil {
			return nil, fmt.Errorf("following redirect: %w", err)
		}
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("closing response body: %w", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, body)
	}
	return io.ReadAll(resp.Body)
}
