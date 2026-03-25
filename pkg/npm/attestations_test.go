package npm

import (
	"context"
	"testing"
	"time"

	"git.duti.dev/secure-package-registry/pkg/config"
)

func TestGetAttestation(t *testing.T) {
	cfg := config.NPMConfig{
		HTTPTimeout: 10 * time.Second,
	}
	client := NewClient(cfg)

	ctx := context.Background()

	// semver@7.6.3 has attestations (published with provenance)
	att, err := client.GetAttestation(ctx, "semver", "7.6.3")
	if err != nil {
		t.Fatalf("GetAttestation failed: %v", err)
	}
	if !att.HasAttestation {
		t.Error("Expected semver@7.6.3 to have attestation")
	}

	// lodash@4.17.21 was published before npm provenance existed
	att2, err := client.GetAttestation(ctx, "lodash", "4.17.21")
	if err != nil {
		t.Fatalf("GetAttestation failed: %v", err)
	}
	if att2.HasAttestation {
		t.Error("Expected lodash@4.17.21 to NOT have attestation")
	}
}

func TestGetAttestationNotFound(t *testing.T) {
	cfg := config.NPMConfig{
		HTTPTimeout: 10 * time.Second,
	}
	client := NewClient(cfg)

	ctx := context.Background()

	// non-existent package should return no attestation
	att, err := client.GetAttestation(ctx, "non-existent-pkg-xyz", "1.0.0")
	if err != nil {
		t.Fatalf("GetAttestation failed: %v", err)
	}
	if att.HasAttestation {
		t.Error("Expected non-existent package to NOT have attestation")
	}
}
