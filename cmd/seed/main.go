// seed populates the database with mock package data for demo purposes.
package main

import (
	"context"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"git.duti.dev/secure-package-registry/internal/gen/coredb"
	"git.duti.dev/secure-package-registry/pkg/logger"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

var log zerolog.Logger

func main() {
	ctx := context.Background()
	log = logger.WithComponent("seed")

	// Database connection string
	databaseURL := getEnv("DATABASE_URL", "postgres://postgres:postgres@core_db:5432/core?sslmode=disable")

	// Connect to database
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		log.Err(err).Msgf("Failed to ping database")
		os.Exit(1)
	}

	fmt.Println("Connected to database successfully")

	queries := coredb.New(pool)

	authUserID, err := seedAuthExamples(ctx, queries)
	if err != nil {
		log.Err(err).Msgf("Failed to seed auth example")
		os.Exit(1)
	}
	log.Info().Msgf("Seeded Auth User. UserID: %s", authUserID)

	fmt.Println("Successfully seeded database with mock data")
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func generateAuthToken() (string, string) {
	b := make([]byte, 32)
	_, err := crand.Read(b)
	if err != nil {
		log.Err(err).Msgf("Failed to generate random bytes for auth token")
	}

	token := hex.EncodeToString(b)

	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])

	return token, tokenHash
}

func seedAuthExamples(ctx context.Context, queries *coredb.Queries) (string, error) {
	// Generate example token and hash
	token, tokenHash := generateAuthToken()

	log.Info().Msgf("User Token: %s", token)

	userid, err := queries.InsertUser(ctx, coredb.InsertUserParams{
		ID:   "seed-user-1",
		Name: "seed-user",
	})
	if err != nil {
		return "", err
	}

	_, err = queries.InsertUserToken(ctx, coredb.InsertUserTokenParams{
		UserID:    userid, // Set actual
		TokenHash: tokenHash,
		ExpiresAt: pgtype.Timestamptz{
			Time:  time.Now().Add(7 * 24 * time.Hour),
			Valid: true,
		},
	})
	if err != nil {
		log.Err(err).Msgf("Failed to insert user token")
	}

	return userid, nil
}
