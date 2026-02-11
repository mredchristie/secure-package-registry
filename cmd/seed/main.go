// seed populates the database with mock package data for demo purposes.
package main

import (
	"context"
	"fmt"
	"os"

	"git.duti.dev/secure-package-registry/internal/gen/coredb"
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

	// Seed packages
	if err := seedPackages(ctx, queries); err != nil {
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

// PackageSeed represents a package to seed
type PackageSeed struct {
	Identifier    string
	Ecosystem     string
	LatestVersion string
	SourceURL     string
}

// seedPackages inserts mock package data
func seedPackages(ctx context.Context, queries *coredb.Queries) error {
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
		{Identifier: "@private/synthetic-search-mcp", Ecosystem: "npm", LatestVersion: "0.2.1", SourceURL: "https://github.com/anomalyco/synthetic-search-mcp"},
	}

	for _, pkg := range packages {
		// Insert package
		packageID, err := queries.InsertPackage(ctx, coredb.InsertPackageParams{
			Identifier:    pkg.Identifier,
			Ecosystem:     coredb.Ecosystem(pkg.Ecosystem),
			LatestVersion: pkg.LatestVersion,
		})
		if err != nil {
			return fmt.Errorf("failed to insert package %s: %w", pkg.Identifier, err)
		}
		fmt.Printf("Inserted package: %s@%s (ID: %d)\n", pkg.Identifier, pkg.LatestVersion, packageID)

		// Insert package version
		err = queries.InsertPackageVersion(ctx, coredb.InsertPackageVersionParams{
			PackageID: packageID,
			Version:   pkg.LatestVersion,
			SourceUrl: pkg.SourceURL,
		})
		if err != nil {
			return fmt.Errorf("failed to insert version for %s: %w", pkg.Identifier, err)
		}
	}

	return nil
}
