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
			URL:    versionRow.SourceUrl,
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

// isNotFoundError checks if the error indicates no rows were found
func isNotFoundError(err error) bool {
	return err == sql.ErrNoRows || err.Error() == "no rows in result set"
}
