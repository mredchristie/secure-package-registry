package pkgdb

import (
	"context"
	"database/sql"
	"fmt"

	"git.duti.dev/secure-package-registry/internal/gen/coredb"
	"github.com/jackc/pgx/v5/pgtype"
)

// PackageVersion represents a package version with metadata and tags
type PackageVersion struct {
	Identifier      string    `json:"identifier"`
	Ecosystem       Ecosystem `json:"ecosystem" enum:"npm,go,cargo,pypi"`
	Version         string    `json:"version"`
	Latest          bool      `json:"latest"`
	Source          Source    `json:"source"`
	TrustLevel      int32     `json:"trust_level"`
	MaintainerNotes string    `json:"maintainer_notes"`
	Tags            []TagInfo `json:"tags"`
}

// Source contains source control information for the package version
type Source struct {
	URL    string `json:"url"`
	Tag    string `json:"tag"`
	Commit string `json:"commit"`
}

// TagInfo holds a tag's value along with its expected type
type TagInfo struct {
	Label     string          `json:"label"`
	ValueType coredb.PkgVtype `json:"value_type" enum:"integer,boolean,float"`
	Data      []byte          `json:"data"`
}

// Client provides a friendly API for querying package version data
type Client struct {
	queries *coredb.Queries
}

// NewClient creates a new package version client
func NewClient(db coredb.DBTX) *Client {
	return &Client{
		queries: coredb.New(db),
	}
}

// GetPackageVersion retrieves package version data for the given ecosystem, identifier, and version
func (c *Client) GetPackageVersion(ctx context.Context, ecosystem Ecosystem, identifier, version string) (*PackageVersion, error) {
	// Fetch package version info
	versionRow, err := c.queries.GetPackageVersion(ctx, coredb.GetPackageVersionParams{
		Ecosystem:  coredb.Ecosystem(ecosystem),
		Identifier: identifier,
		Version:    version,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch package version: %w", err)
	}

	// Fetch tags
	queryParams := coredb.GetPackageVersionTagsParams{
		Ecosystem:  coredb.Ecosystem(ecosystem),
		Identifier: identifier,
		Version:    version,
	}
	tagRows, err := c.queries.GetPackageVersionTags(ctx, queryParams)
	if err != nil && !isNotFoundError(err) {
		return nil, fmt.Errorf("failed to fetch package tags: %w", err)
	}

	// Build tags list
	tagsList := make([]TagInfo, 0, len(tagRows))
	for _, row := range tagRows {
		if len(row.Value) > 0 && string(row.Value) != "null" {
			tagsList = append(tagsList, TagInfo{
				Label:     row.Label,
				ValueType: row.ValueType,
				Data:      row.Value,
			})
		}
	}

	// Construct PackageVersion
	pv := &PackageVersion{
		Identifier: versionRow.Identifier,
		Ecosystem:  Ecosystem(versionRow.PEcosystem),
		Version:    versionRow.Version,
		Latest:     versionRow.Latest,
		Source: Source{
			URL:    pgtypeTextToString(versionRow.SourceUrl),
			Tag:    pgtypeTextToString(versionRow.SourceTag),
			Commit: pgtypeTextToString(versionRow.SourceCommitHash),
		},
		TrustLevel:      pgtypeInt4ToInt32(versionRow.TrustLevel),
		MaintainerNotes: pgtypeTextToString(versionRow.MaintainerNotes),
		Tags:            tagsList,
	}

	return pv, nil
}

// pgtypeTextToString converts pgtype.Text to string
func pgtypeTextToString(t pgtype.Text) string {
	if t.Valid {
		return t.String
	}
	return ""
}

// pgtypeInt4ToInt32 converts pgtype.Int4 to int32
func pgtypeInt4ToInt32(i pgtype.Int4) int32 {
	if i.Valid {
		return i.Int32
	}
	return 0
}

// VersionSummary represents a version in a version list with verification status
type VersionSummary struct {
	Version          string  `json:"version"`
	Latest           bool    `json:"latest"`
	Source           Source  `json:"source"`
	HasAttestation   bool    `json:"has_attestation"`
	HasOSSRebuild    bool    `json:"has_oss_rebuild"`
	BehaviorPassed   bool    `json:"behavior_passed"`
	ManuallyApproved *bool   `json:"manually_approved"`
	ReviewComment    *string `json:"review_comment"`
}

// VersionListResult holds a list of version summaries for a package
type VersionListResult struct {
	Identifier string           `json:"identifier"`
	Ecosystem  Ecosystem        `json:"ecosystem" enum:"npm,go,cargo,pypi"`
	Versions   []VersionSummary `json:"versions"`
}

// boolPtrFromJSONB converts a JSONB value to *bool.
// Returns nil if the input is nil or not a valid boolean.
func boolPtrFromJSONB(data []byte) *bool {
	if data == nil {
		return nil
	}
	s := string(data)
	if s == "true" {
		return func() *bool { v := true; return &v }()
	}
	if s == "false" {
		return func() *bool { v := false; return &v }()
	}
	return nil
}

// stringPtrFromJSONB converts a JSONB value to *string.
// Returns nil if the input is nil.
// JSONB text values are stored as quoted strings, so we strip the outer quotes.
func stringPtrFromJSONB(data []byte) *string {
	if data == nil {
		return nil
	}
	s := string(data)
	// JSONB text values are stored as "quoted" strings
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	return &s
}

// ListVersionsPublic returns all versions for a package with their verification tags
func (c *Client) ListVersionsPublic(ctx context.Context, ecosystem Ecosystem, identifier string) (*VersionListResult, error) {
	rows, err := c.queries.ListPackageVersionsPublic(ctx, coredb.ListPackageVersionsPublicParams{
		Ecosystem:  coredb.Ecosystem(ecosystem),
		Identifier: identifier,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list versions: %w", err)
	}

	versions := make([]VersionSummary, 0, len(rows))
	for _, row := range rows {
		versions = append(versions, VersionSummary{
			Version: row.Version,
			Latest:  row.Latest,
			Source: Source{
				URL:    pgtypeTextToString(row.SourceUrl),
				Tag:    pgtypeTextToString(row.SourceTag),
				Commit: pgtypeTextToString(row.SourceCommitHash),
			},
			HasAttestation:   row.HasAttestation,
			HasOSSRebuild:    row.HasOssRebuild,
			BehaviorPassed:   row.BehaviorPassed,
			ManuallyApproved: boolPtrFromJSONB(row.ManuallyApproved),
			ReviewComment:    stringPtrFromJSONB(row.ReviewComment),
		})
	}

	return &VersionListResult{
		Identifier: identifier,
		Ecosystem:  ecosystem,
		Versions:   versions,
	}, nil
}

// isNotFoundError checks if the error indicates no rows were found
func isNotFoundError(err error) bool {
	return err == sql.ErrNoRows || err.Error() == "no rows in result set"
}
