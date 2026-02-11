package pkgdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"git.duti.dev/secure-package-registry/internal/gen/coredb"
	"github.com/jackc/pgx/v5/pgtype"
)

// PackageVersion represents a package version with metadata and tags
type PackageVersion struct {
	Identifier      string             `json:"identifier"`
	Ecosystem       Ecosystem          `json:"ecosystem" enum:"npm,go,cargo,pypi"`
	Version         string             `json:"version"`
	Latest          bool               `json:"latest"`
	Source          Source             `json:"source"`
	TrustLevel      int32              `json:"trust_level"`
	MaintainerNotes string             `json:"maintainer_notes"`
	Tags            PackageVersionTags `json:"tags"`
}

// Source contains source control information for the package version
type Source struct {
	URL    string `json:"url"`
	Tag    string `json:"tag"`
	Commit string `json:"commit"`
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

	// Build tags map with type validation
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
		Tags: PackageVersionTags{
			Tags: tagsList,
		},
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

// PackageVersionTags provides typed accessors for tag values
type PackageVersionTags struct {
	Tags []TagInfo
}

// TagInfo holds a tag's value along with its expected type
type TagInfo struct {
	Label     string          `json:"label"`
	ValueType coredb.PkgVtype `json:"value_type" enum:"integer,boolean,float"`
	Data      []byte          `json:"data"`
}

// IsTrusted returns whether the package version is trusted
func (t PackageVersionTags) IsTrusted() (bool, error) {
	return t.getBool("is_trusted")
}

// IsReproducible returns whether the package build is reproducible
func (t PackageVersionTags) IsReproducible() (bool, error) {
	return t.getBool("reproducible")
}

// IsBehaviorClean returns whether the package has clean runtime behavior
func (t PackageVersionTags) IsBehaviorClean() (bool, error) {
	return t.getBool("behavior_clean")
}

// GetCVECount returns the number of known CVEs
func (t PackageVersionTags) GetCVECount() (int32, error) {
	return t.getInt32("cve_count")
}

// GetAnomalyScore returns the behavioral anomaly score (0.0-1.0)
func (t PackageVersionTags) GetAnomalyScore() (float32, error) {
	return t.getFloat32("anomaly_score")
}

// getBool unmarshals a boolean tag value
func (t PackageVersionTags) getBool(key string) (bool, error) {
	tag, err := t.findTag(key)
	if err != nil {
		return false, err
	}

	// Validate expected type
	if tag.ValueType != coredb.PkgVtypeBoolean {
		return false, &TagError{
			Kind: KindParseFailed,
			Tag:  key,
			Err:  fmt.Errorf("expected boolean, got %s", tag.ValueType),
		}
	}

	var val bool
	if err := t.unmarshalTag(key, tag.Data, &val); err != nil {
		return false, &TagError{Kind: KindParseFailed, Tag: key, Err: err}
	}
	return val, nil
}

// getInt32 unmarshals an int32 tag value
func (t PackageVersionTags) getInt32(key string) (int32, error) {
	tag, err := t.findTag(key)
	if err != nil {
		return 0, err
	}

	// Validate expected type
	if tag.ValueType != coredb.PkgVtypeInteger {
		return 0, &TagError{
			Kind: KindParseFailed,
			Tag:  key,
			Err:  fmt.Errorf("expected integer, got %s", tag.ValueType),
		}
	}

	var val int32
	if err := t.unmarshalTag(key, tag.Data, &val); err != nil {
		return 0, &TagError{Kind: KindParseFailed, Tag: key, Err: err}
	}
	return val, nil
}

// getFloat32 unmarshals a float32 tag value
func (t PackageVersionTags) getFloat32(key string) (float32, error) {
	tag, err := t.findTag(key)
	if err != nil {
		return 0, err
	}

	// Validate expected type
	if tag.ValueType != coredb.PkgVtypeFloat {
		return 0, &TagError{
			Kind: KindParseFailed,
			Tag:  key,
			Err:  fmt.Errorf("expected float, got %s", tag.ValueType),
		}
	}

	var val float32
	if err := t.unmarshalTag(key, tag.Data, &val); err != nil {
		return 0, &TagError{Kind: KindParseFailed, Tag: key, Err: err}
	}
	return val, nil
}

// findTag locates a tag by label
func (t PackageVersionTags) findTag(key string) (*TagInfo, error) {
	if t.Tags == nil {
		return nil, fmt.Errorf("tags list is nil")
	}

	for _, tag := range t.Tags {
		if tag.Label == key {
			return &tag, nil
		}
	}

	return nil, &TagError{Kind: KindNotFound, Tag: key}
}

// unmarshalTag unmarshals a tag value to the provided target
func (t PackageVersionTags) unmarshalTag(key string, data []byte, target interface{}) error {
	// Check if value is null/empty
	if len(data) == 0 || string(data) == "null" {
		return &TagError{Kind: KindNoValue, Tag: key}
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal tag %q: %w", key, err)
	}
	return nil
}
