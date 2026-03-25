package ossrebuild

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"git.duti.dev/secure-package-registry/pkg/config"
	"git.duti.dev/secure-package-registry/pkg/logger"

	"github.com/rs/zerolog"
)

const (
	gcsBaseURL = "https://storage.googleapis.com/google-rebuild-attestations"
)

type Client struct {
	httpClient *http.Client
	log        zerolog.Logger
}

func NewClient(cfg config.OSSRebuildConfig) *Client {
	transport := cfg.HTTPTransport
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &Client{
		httpClient: &http.Client{
			Timeout:   cfg.HTTPTimeout,
			Transport: transport,
		},
		log: logger.WithComponent("oss-rebuild-client"),
	}
}

func (c *Client) CheckAttestation(ctx context.Context, eco Ecosystem, pkg, version string) (*Attestation, error) {
	url := c.buildURL(eco, pkg, version)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			c.log.Warn().Err(cerr).Msg("Failed to close response body")
		}
	}()

	// Object doesn't exist - no attestation
	if resp.StatusCode == http.StatusNotFound {
		return &Attestation{
			Ecosystem:      eco,
			Package:        pkg,
			Version:        version,
			HasAttestation: false,
		}, nil
	}

	// Unexpected status code - treat as error
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	return &Attestation{
		Ecosystem:         eco,
		Package:           pkg,
		Version:           version,
		HasAttestation:    true,
		AttestationBundle: body,
	}, nil
}

func (c *Client) buildURL(eco Ecosystem, pkg, version string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s-%s.tgz/rebuild.intoto.jsonl",
		gcsBaseURL, eco, pkg, version, pkg, version)
}

func DefaultClient() *Client {
	return NewClient(config.OSSRebuildConfig{
		HTTPTimeout: 10 * time.Second,
	})
}
