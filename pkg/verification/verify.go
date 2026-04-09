// Package verification provides shared logic for checking upstream attestation
// and OSS rebuild status of package versions, persisting the results as tags.
package verification

import (
	"context"
	"fmt"

	"git.duti.dev/secure-package-registry/internal/gen/coredb"
	"git.duti.dev/secure-package-registry/pkg/logger"
	"git.duti.dev/secure-package-registry/pkg/npm"
	ossrebuild "git.duti.dev/secure-package-registry/pkg/oss-rebuild"

	"github.com/rs/zerolog"
)

// Result holds the outcome of a verification check.
type Result struct {
	Ecosystem           string `json:"ecosystem"`
	Identifier          string `json:"identifier"`
	Version             string `json:"version"`
	UpstreamAttestation bool   `json:"upstream_attestation"`
	OSSRebuild          bool   `json:"oss_rebuild"`
}

// Service checks upstream attestation and OSS rebuild status for package
// versions and persists the results as database tags.
type Service struct {
	queries   coredb.Querier
	npmClient *npm.Client
	ossClient *ossrebuild.Client
	log       zerolog.Logger
}

// NewService creates a verification service.
func NewService(queries coredb.Querier, npmClient *npm.Client, ossClient *ossrebuild.Client) *Service {
	return &Service{
		queries:   queries,
		npmClient: npmClient,
		ossClient: ossClient,
		log:       logger.WithComponent("verification"),
	}
}

// CheckAndTag checks upstream attestation and OSS rebuild for the given
// package version, persists the results as tags, and returns the outcome.
func (s *Service) CheckAndTag(ctx context.Context, ecosystem, identifier, version string) (*Result, error) {
	if ecosystem != string(coredb.EcosystemNpm) {
		return nil, fmt.Errorf("unsupported ecosystem %q (only npm is supported)", ecosystem)
	}

	// Look up the package version to confirm it exists.
	pvID, err := s.queries.GetPackageVersionID(ctx, coredb.GetPackageVersionIDParams{
		Ecosystem:  coredb.Ecosystem(ecosystem),
		Identifier: identifier,
		Version:    version,
	})
	if err != nil {
		return nil, fmt.Errorf("looking up package version: %w", err)
	}

	// Run both checks concurrently.
	type attestResult struct {
		has bool
		err error
	}
	type ossResult struct {
		has bool
		err error
	}

	attestCh := make(chan attestResult, 1)
	ossCh := make(chan ossResult, 1)

	go func() {
		att, err := s.npmClient.GetAttestation(ctx, identifier, version)
		if err != nil {
			attestCh <- attestResult{err: err}
			return
		}
		attestCh <- attestResult{has: att.HasAttestation}
	}()

	go func() {
		att, err := s.ossClient.CheckNPMContext(ctx, identifier, version)
		if err != nil {
			ossCh <- ossResult{err: err}
			return
		}
		ossCh <- ossResult{has: att.HasAttestation}
	}()

	attestRes := <-attestCh
	ossRes := <-ossCh

	// Log errors but don't fail — we still persist whatever we got.
	if attestRes.err != nil {
		s.log.Warn().Err(attestRes.err).
			Str("identifier", identifier).
			Str("version", version).
			Msg("npm attestation check failed")
	}
	if ossRes.err != nil {
		s.log.Warn().Err(ossRes.err).
			Str("identifier", identifier).
			Str("version", version).
			Msg("OSS rebuild check failed")
	}

	// Persist tags — only if the respective check succeeded (no error).
	if attestRes.err == nil {
		if err := s.upsertBoolTag(ctx, pvID, "upstream_attestation", attestRes.has); err != nil {
			s.log.Error().Err(err).Msg("Failed to upsert upstream_attestation tag")
		}
	}
	if ossRes.err == nil {
		if err := s.upsertBoolTag(ctx, pvID, "oss_rebuild", ossRes.has); err != nil {
			s.log.Error().Err(err).Msg("Failed to upsert oss_rebuild tag")
		}
	}

	return &Result{
		Ecosystem:           ecosystem,
		Identifier:          identifier,
		Version:             version,
		UpstreamAttestation: attestRes.has,
		OSSRebuild:          ossRes.has,
	}, nil
}

// upsertBoolTag sets a boolean tag on a package version.
func (s *Service) upsertBoolTag(ctx context.Context, pvID int32, label string, value bool) error {
	tagTypeID, err := s.queries.GetTagTypeByLabel(ctx, label)
	if err != nil {
		return fmt.Errorf("looking up tag type %q: %w", label, err)
	}

	val := []byte(`false`)
	if value {
		val = []byte(`true`)
	}

	if err := s.queries.InsertPackageTag(ctx, coredb.InsertPackageTagParams{
		PackageVersion: pvID,
		TagType:        tagTypeID,
		Value:          val,
	}); err != nil {
		return fmt.Errorf("upserting tag %q: %w", label, err)
	}

	return nil
}
