// Package github provides a client for the GitHub Actions API, scoped to
// workflow dispatch, run polling, and artifact download.
package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"git.duti.dev/secure-package-registry/pkg/logger"

	"github.com/rs/zerolog"
)

const (
	baseURL    = "https://api.github.com"
	apiVersion = "2022-11-28"
)

// Config holds the parameters needed to construct a Client.
type Config struct {
	Token string
	Owner string
	Repo  string
	// HTTPTransport overrides the default transport, enabling test injection (e.g. VCR).
	HTTPTransport http.RoundTripper
}

// Client is a GitHub Actions API client scoped to a single repository.
type Client struct {
	owner      string
	repo       string
	token      string
	httpClient *http.Client
	log        zerolog.Logger
}

// NewClient creates a Client from the given Config.
func NewClient(cfg Config) *Client {
	transport := cfg.HTTPTransport
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &Client{
		owner: cfg.Owner,
		repo:  cfg.Repo,
		token: cfg.Token,
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
		log: logger.WithComponent("github-client"),
	}
}

// doRequest executes an authenticated request against the repo-scoped API.
// path is relative to /repos/{owner}/{repo}/ (e.g. "actions/runs/123").
func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader) (resp *http.Response, err error) {
	url := fmt.Sprintf("%s/repos/%s/%s/%s", baseURL, c.owner, c.repo, path)
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("X-GitHub-Api-Version", apiVersion)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.httpClient.Do(req)
}

// doJSON executes an authenticated request and decodes the JSON response into target.
func (c *Client) doJSON(ctx context.Context, method, path string, body io.Reader, target any) (err error) {
	resp, err := c.doRequest(ctx, method, path, body)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("closing response body: %w", closeErr)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, respBody)
	}
	if err = json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	return nil
}

// postJSON marshals payload as JSON and calls doJSON with a POST request.
func (c *Client) postJSON(ctx context.Context, path string, payload any, target any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}
	return c.doJSON(ctx, http.MethodPost, path, bytes.NewReader(data), target)
}
