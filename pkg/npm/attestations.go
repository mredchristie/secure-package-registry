package npm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	attestationsURL = "https://registry.npmjs.org/-/npm/v1/attestations"
)

type Attestation struct {
	Package        string
	Version        string
	HasAttestation bool
}

type attestationResponse struct {
	Attestations []attestationEntry `json:"attestations"`
}

type attestationEntry struct {
	PredicateType string          `json:"predicateType"`
	Bundle        json.RawMessage `json:"bundle"`
}

func (c *Client) GetAttestation(ctx context.Context, name, version string) (*Attestation, error) {
	url := fmt.Sprintf("%s/%s@%s", attestationsURL, name, version)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			c.log.Warn().Err(cerr).Msg("Failed to close response body")
		}
	}()

	// No attestation exists for this package version
	if resp.StatusCode == http.StatusNotFound {
		return &Attestation{
			Package:        name,
			Version:        version,
			HasAttestation: false,
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}

	var result attestationResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	// Check if there are any attestations (publish or provenance)
	hasAttestation := len(result.Attestations) > 0

	return &Attestation{
		Package:        name,
		Version:        version,
		HasAttestation: hasAttestation,
	}, nil
}
