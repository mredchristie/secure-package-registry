package pkgdb

import (
	"context"
	"fmt"

	"git.duti.dev/secure-package-registry/internal/gen/coredb"
	"github.com/jackc/pgx/v5/pgtype"
)

// PackageSummary represents a package in search results
type PackageSummary struct {
	Identifier    string    `json:"identifier"`
	Ecosystem     Ecosystem `json:"ecosystem" enum:"npm,go,cargo,pypi"`
	LatestVersion string    `json:"latest_version"`
}

// SearchParams represents parameters for searching packages
type SearchParams struct {
	Query     string
	Ecosystem *Ecosystem
	Page      int32
	PageSize  int32
}

// SearchResult represents a paginated search response
type SearchResult struct {
	Items      []PackageSummary `json:"items"`
	TotalCount int64            `json:"total_count"`
	Page       int32            `json:"page"`
	PageSize   int32            `json:"page_size"`
}

// SearchPackages searches for packages matching the query
func (c *Client) SearchPackages(ctx context.Context, params SearchParams) (*SearchResult, error) {
	// Build ecosystem parameter
	var ecosystemParam coredb.NullEcosystem
	if params.Ecosystem != nil {
		ecosystemParam = coredb.NullEcosystem{
			Ecosystem: *params.Ecosystem,
			Valid:     true,
		}
	}

	queryParam := pgtype.Text{String: params.Query, Valid: true}

	// Execute count query
	totalCount, err := c.queries.CountSearchPackages(ctx, coredb.CountSearchPackagesParams{
		Query:     queryParam,
		Ecosystem: ecosystemParam,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to count search results: %w", err)
	}

	// Execute search query
	rows, err := c.queries.SearchPackages(ctx, coredb.SearchPackagesParams{
		Query:     queryParam,
		Ecosystem: ecosystemParam,
		Page:      params.Page,
		PageSize:  params.PageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search packages: %w", err)
	}

	// Build result
	items := make([]PackageSummary, 0, len(rows))
	for _, row := range rows {
		var latestVersion string
		if row.LatestVersion.Valid {
			latestVersion = row.LatestVersion.String
		}
		items = append(items, PackageSummary{
			Identifier:    row.Identifier,
			Ecosystem:     Ecosystem(row.PEcosystem),
			LatestVersion: latestVersion,
		})
	}

	return &SearchResult{
		Items:      items,
		TotalCount: totalCount,
		Page:       params.Page,
		PageSize:   params.PageSize,
	}, nil
}
