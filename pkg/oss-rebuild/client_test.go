package ossrebuild

import (
	"context"
	"testing"
	"time"

	"git.duti.dev/secure-package-registry/pkg/config"
)

func TestCheckAttestation(t *testing.T) {
	cfg := config.OSSRebuildConfig{
		HTTPTimeout: 10 * time.Second,
	}
	client := NewClient(cfg)

	ctx := context.Background()

	// semver@7.6.2 has an oss-rebuild attestation
	att, err := client.CheckAttestation(ctx, EcosystemNPM, "semver", "7.6.2")
	if err != nil {
		t.Fatalf("CheckAttestation failed: %v", err)
	}
	if !att.HasAttestation {
		t.Error("Expected semver@7.6.2 to have oss-rebuild attestation")
	}

	// lodash is not in oss-rebuild (not popular enough)
	att2, err := client.CheckAttestation(ctx, EcosystemNPM, "lodash", "4.17.21")
	if err != nil {
		t.Fatalf("CheckAttestation failed: %v", err)
	}
	if att2.HasAttestation {
		t.Error("Expected lodash@4.17.21 to NOT have oss-rebuild attestation")
	}
}
