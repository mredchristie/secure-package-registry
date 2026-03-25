package ossrebuild

import (
	"context"
)

func (c *Client) CheckNPM(pkg, version string) (*Attestation, error) {
	return c.CheckAttestation(context.Background(), EcosystemNPM, pkg, version)
}

func (c *Client) CheckNPMContext(ctx context.Context, pkg, version string) (*Attestation, error) {
	return c.CheckAttestation(ctx, EcosystemNPM, pkg, version)
}
