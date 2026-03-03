// seed populates the database with mock package data for demo purposes.
package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"

	"git.duti.dev/secure-package-registry/internal/gen/coredb"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()

	// Database connection string
	databaseURL := getEnv("DATABASE_URL", "postgres://core:PleaseChangeMe@core_db:5432/secure_registry?sslmode=disable")

	// Connect to database
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to ping database: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Connected to database successfully")

	queries := coredb.New(pool)

	// Seed tag types first
	tagTypeIDs, err := seedTagTypes(ctx, queries)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to seed tag types: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Seeded %d tag types\n", len(tagTypeIDs))

	// Seed packages with versions and tags
	if err := seedPackages(ctx, queries, tagTypeIDs); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to seed packages: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Successfully seeded database with mock data")
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// TagTypeSeed represents a tag type to seed
type TagTypeSeed struct {
	ID          int32
	Label       string
	Description string
	ValueType   string
}

// Hardcoded tag types with IDs
var tagTypes = []TagTypeSeed{
	{ID: 1, Label: "is_trusted", Description: "Package is trusted", ValueType: "boolean"},
	{ID: 2, Label: "reproducible", Description: "Build matches published artifact", ValueType: "boolean"},
	{ID: 3, Label: "behavior_clean", Description: "No suspicious runtime behavior", ValueType: "boolean"},
	{ID: 4, Label: "cve_count", Description: "Number of known CVEs", ValueType: "integer"},
	{ID: 5, Label: "anomaly_score", Description: "Behavioral anomaly severity 0.0-1.0", ValueType: "float"},
}

// seedTagTypes inserts tag types and returns their IDs
func seedTagTypes(ctx context.Context, queries *coredb.Queries) (map[string]int32, error) {
	tagTypeIDs := make(map[string]int32)

	for _, tt := range tagTypes {
		id, err := queries.InsertTagType(ctx, coredb.InsertTagTypeParams{
			Label:       tt.Label,
			Description: pgtype.Text{String: tt.Description, Valid: true},
			ValueType:   coredb.PkgVtype(tt.ValueType),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to insert tag type %s: %w", tt.Label, err)
		}
		tagTypeIDs[tt.Label] = id
		fmt.Printf("Inserted tag type: %s (ID: %d)\n", tt.Label, id)
	}

	return tagTypeIDs, nil
}

// PackageSeed represents a package to seed
type PackageSeed struct {
	Identifier    string
	Ecosystem     string
	LatestVersion string
	SourceURL     string
	IsTrusted     bool // Special flag for synthetic-search-mcp
}

// seedPackages inserts mock package data with tags
func seedPackages(ctx context.Context, queries *coredb.Queries, tagTypeIDs map[string]int32) error {
	// React-related packages for demo
	packages := []PackageSeed{
		{Identifier: "react", Ecosystem: "npm", LatestVersion: "18.2.0", SourceURL: "https://github.com/facebook/react"},
		{Identifier: "react-dom", Ecosystem: "npm", LatestVersion: "18.2.0", SourceURL: "https://github.com/facebook/react"},
		{Identifier: "react-router", Ecosystem: "npm", LatestVersion: "6.20.0", SourceURL: "https://github.com/remix-run/react-router"},
		{Identifier: "react-router-dom", Ecosystem: "npm", LatestVersion: "6.20.0", SourceURL: "https://github.com/remix-run/react-router"},
		{Identifier: "react-query", Ecosystem: "npm", LatestVersion: "3.39.3", SourceURL: "https://github.com/TanStack/query"},
		{Identifier: "react-hook-form", Ecosystem: "npm", LatestVersion: "7.48.0", SourceURL: "https://github.com/react-hook-form/react-hook-form"},
		{Identifier: "react-spring", Ecosystem: "npm", LatestVersion: "9.7.3", SourceURL: "https://github.com/pmndrs/react-spring"},
		{Identifier: "react-use", Ecosystem: "npm", LatestVersion: "17.4.0", SourceURL: "https://github.com/streamich/react-use"},
		{Identifier: "react-testing-library", Ecosystem: "npm", LatestVersion: "14.1.0", SourceURL: "https://github.com/testing-library/react-testing-library"},
		{Identifier: "synthetic-search-mcp", Ecosystem: "npm", LatestVersion: "0.2.1", SourceURL: "https://github.com/anomalyco/synthetic-search-mcp", IsTrusted: true},
	}

	for _, pkg := range packages {
		// Insert package
		packageID, err := queries.InsertPackage(ctx, coredb.InsertPackageParams{
			Identifier:    pkg.Identifier,
			Ecosystem:     coredb.Ecosystem(pkg.Ecosystem),
			LatestVersion: pgtype.Text{String: pkg.LatestVersion, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to insert package %s: %w", pkg.Identifier, err)
		}
		fmt.Printf("Inserted package: %s@%s (ID: %d)\n", pkg.Identifier, pkg.LatestVersion, packageID)

		// Insert package version and get the version ID
		versionID, err := queries.InsertPackageVersion(ctx, coredb.InsertPackageVersionParams{
			PackageID: packageID,
			Version:   pkg.LatestVersion,
			SourceUrl: pkg.SourceURL,
		})
		if err != nil {
			return fmt.Errorf("failed to insert version for %s: %w", pkg.Identifier, err)
		}

		// Insert tags for this version
		tags := generateTags(pkg.IsTrusted)
		if err := insertTags(ctx, queries, versionID, tagTypeIDs, tags); err != nil {
			return fmt.Errorf("failed to insert tags for %s: %w", pkg.Identifier, err)
		}
		fmt.Printf("  Added tags (version ID: %d)\n", versionID)
	}

	return nil
}

// TagValues represents the tag values for a package
type TagValues struct {
	IsTrusted     bool
	Reproducible  bool
	BehaviorClean bool
	CVECount      int32
	AnomalyScore  float64
}

// Predefined tag combinations for random selection
var tagCombinations = []TagValues{
	{IsTrusted: true, Reproducible: true, BehaviorClean: true, CVECount: 0, AnomalyScore: 0.02},
	{IsTrusted: true, Reproducible: true, BehaviorClean: true, CVECount: 1, AnomalyScore: 0.05},
	{IsTrusted: true, Reproducible: false, BehaviorClean: true, CVECount: 0, AnomalyScore: 0.10},
	{IsTrusted: false, Reproducible: false, BehaviorClean: true, CVECount: 2, AnomalyScore: 0.25},
	{IsTrusted: false, Reproducible: false, BehaviorClean: false, CVECount: 5, AnomalyScore: 0.60},
}

// generateTags creates tag values for a package
func generateTags(isTrusted bool) TagValues {
	if isTrusted {
		// Perfect scores for trusted packages
		return TagValues{
			IsTrusted:     true,
			Reproducible:  true,
			BehaviorClean: true,
			CVECount:      0,
			AnomalyScore:  0.0,
		}
	}
	// Random combination for others
	return tagCombinations[rand.Intn(len(tagCombinations))]
}

// insertTags inserts tag values for a version
func insertTags(ctx context.Context, queries *coredb.Queries, versionID int32, tagTypeIDs map[string]int32, tags TagValues) error {
	// is_trusted
	err := queries.InsertPackageTag(ctx, coredb.InsertPackageTagParams{
		PackageVersion: versionID,
		TagType:        tagTypeIDs["is_trusted"],
		Value:          []byte(boolToJSON(tags.IsTrusted)),
	})
	if err != nil {
		return fmt.Errorf("failed to insert is_trusted tag: %w", err)
	}

	// reproducible
	err = queries.InsertPackageTag(ctx, coredb.InsertPackageTagParams{
		PackageVersion: versionID,
		TagType:        tagTypeIDs["reproducible"],
		Value:          []byte(boolToJSON(tags.Reproducible)),
	})
	if err != nil {
		return fmt.Errorf("failed to insert reproducible tag: %w", err)
	}

	// behavior_clean
	err = queries.InsertPackageTag(ctx, coredb.InsertPackageTagParams{
		PackageVersion: versionID,
		TagType:        tagTypeIDs["behavior_clean"],
		Value:          []byte(boolToJSON(tags.BehaviorClean)),
	})
	if err != nil {
		return fmt.Errorf("failed to insert behavior_clean tag: %w", err)
	}

	// cve_count
	err = queries.InsertPackageTag(ctx, coredb.InsertPackageTagParams{
		PackageVersion: versionID,
		TagType:        tagTypeIDs["cve_count"],
		Value:          []byte(intToJSON(tags.CVECount)),
	})
	if err != nil {
		return fmt.Errorf("failed to insert cve_count tag: %w", err)
	}

	// anomaly_score
	err = queries.InsertPackageTag(ctx, coredb.InsertPackageTagParams{
		PackageVersion: versionID,
		TagType:        tagTypeIDs["anomaly_score"],
		Value:          []byte(floatToJSON(tags.AnomalyScore)),
	})
	if err != nil {
		return fmt.Errorf("failed to insert anomaly_score tag: %w", err)
	}

	return nil
}

// boolToJSON converts bool to JSON string
func boolToJSON(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// intToJSON converts int32 to JSON string
func intToJSON(i int32) string {
	return fmt.Sprintf("%d", i)
}

// floatToJSON converts float64 to JSON string
func floatToJSON(f float64) string {
	return fmt.Sprintf("%.2f", f)
}
