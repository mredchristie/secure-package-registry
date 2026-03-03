package npm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"git.duti.dev/secure-package-registry/pkg/config"
	"git.duti.dev/secure-package-registry/pkg/logger"
	"github.com/rs/zerolog"
)

// Client is a unified HTTP client for the NPM registry and CouchDB changes feed.
// It handles both package metadata fetches (from registry.npmjs.org) and
// changes feed polling (from replicate.npmjs.com).
type Client struct {
	replicateURL string
	registryURL  string
	httpClient   *http.Client
	log          zerolog.Logger
}

// NewClient creates a new NPM client from the given NPM config.
func NewClient(cfg config.NPMConfig) *Client {
	registryURL := cfg.RegistryURL
	if registryURL == "" {
		registryURL = "https://registry.npmjs.org"
	}
	replicateURL := cfg.ReplicateURL
	if replicateURL == "" {
		replicateURL = "https://replicate.npmjs.com"
	}
	return &Client{
		replicateURL: replicateURL,
		registryURL:  registryURL,
		httpClient: &http.Client{
			Timeout:   cfg.HTTPTimeout,
			Transport: cfg.HTTPTransport,
		},
		log: logger.WithComponent("npm-client"),
	}
}

// doJSON performs an HTTP GET request and decodes the JSON response into dest.
// It handles request creation, execution, body closing, and status code checking.
func (c *Client) doJSON(ctx context.Context, url string, headers map[string]string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			c.log.Warn().Err(cerr).Msg("Failed to close response body")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}

	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	return nil
}

// FetchPackageMetadata fetches abbreviated package metadata from the NPM registry.
func (c *Client) FetchPackageMetadata(ctx context.Context, pkgName string) (*NPMPackage, error) {
	u := fmt.Sprintf("%s/%s", c.registryURL, pkgName)
	headers := map[string]string{
		"Accept": "application/vnd.npm.install-v1+json",
	}

	var pkg NPMPackage
	if err := c.doJSON(ctx, u, headers, &pkg); err != nil {
		return nil, fmt.Errorf("fetching package %s: %w", pkgName, err)
	}

	return &pkg, nil
}

func (c *Client) GetLatestVersion(ctx context.Context, pkgName string) (string, error) {
	pkg, err := c.FetchPackageMetadata(ctx, pkgName)
	if err != nil {
		return "", fmt.Errorf("fetching package metadata: %w", err)
	}

	latest, ok := pkg.DistTags["latest"]
	if !ok {
		return "", fmt.Errorf("latest version not found for package %s", pkgName)
	}

	return latest, nil
}

// GetRegistryInfo fetches current registry metadata including update_seq.
func (c *Client) GetRegistryInfo(ctx context.Context) (*RegistryInfo, error) {
	var info RegistryInfo
	if err := c.doJSON(ctx, c.replicateURL, nil, &info); err != nil {
		return nil, fmt.Errorf("fetching registry info: %w", err)
	}

	return &info, nil
}

// PollChanges fetches changes since the given sequence.
// Uses the default (batch) feed mode since NPM doesn't support continuous.
func (c *Client) PollChanges(ctx context.Context, since SequenceID, limit int) (*ChangesResponse, error) {
	u, err := url.Parse(c.replicateURL + "/_changes")
	if err != nil {
		return nil, fmt.Errorf("parsing changes URL: %w", err)
	}

	q := u.Query()
	if since.value != "" {
		q.Set("since", since.value)
	}
	if limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}
	u.RawQuery = q.Encode()

	c.log.Debug().Str("url", u.String()).Msg("Polling changes")

	var result ChangesResponse
	if err := c.doJSON(ctx, u.String(), nil, &result); err != nil {
		return nil, fmt.Errorf("polling changes: %w", err)
	}

	return &result, nil
}
