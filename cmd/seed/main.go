// seed populates the database with a BetterAuth-compatible user and API key for local dev.
package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"git.duti.dev/secure-package-registry/internal/gen/coredb"
	"git.duti.dev/secure-package-registry/pkg/logger"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

var log zerolog.Logger

func main() {
	ctx := context.Background()
	log = logger.WithComponent("seed")

	databaseURL := getEnv("DATABASE_URL", "postgres://postgres:postgres@core_db:5432/core?sslmode=disable")

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Err(err).Msg("Failed to ping database")
		os.Exit(1)
	}

	fmt.Println("Connected to database successfully")

	queries := coredb.New(pool)

	rawKey, err := seedUserAndAPIKey(ctx, pool, queries)
	if err != nil {
		log.Err(err).Msg("Failed to seed user and API key")
		os.Exit(1)
	}

	fmt.Printf("API Key (use as Bearer token): %s\n", rawKey)
	fmt.Println("Successfully seeded database with dev data")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// seedUserAndAPIKey creates a BetterAuth user and a BetterAuth-compatible API key row.
// Returns the raw API key (the token to use as a Bearer token).
func seedUserAndAPIKey(ctx context.Context, pool *pgxpool.Pool, queries *coredb.Queries) (string, error) {
	const userID = "seed-user-1"
	const userName = "seed-user"

	_, err := queries.InsertUser(ctx, coredb.InsertUserParams{
		ID:   userID,
		Name: userName,
	})
	if err != nil {
		log.Warn().Err(err).Msg("User may already exist, continuing")
	} else {
		log.Info().Str("user_id", userID).Msg("Seeded user")
	}

	// Generate a random 32-byte API key and encode as hex (64 chars).
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		return "", fmt.Errorf("generating random bytes: %w", err)
	}
	rawKey := hex.EncodeToString(rawBytes)

	// Hash the same way BetterAuth does: SHA-256 -> base64url no-padding.
	hash := sha256.Sum256([]byte(rawKey))
	keyHash := base64.RawURLEncoding.EncodeToString(hash[:])

	// Generate an ID for the apikey row.
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return "", fmt.Errorf("generating apikey id: %w", err)
	}
	apikeyID := hex.EncodeToString(idBytes)

	// Insert directly into the apikey table.
	// We use the pool directly since sqlc doesn't have an insert query for apikey.
	_, err = pool.Exec(ctx, `
		INSERT INTO apikey ("id", "name", "start", "prefix", "key", "configId", "referenceId", "enabled", "createdAt", "updatedAt")
		VALUES ($1, $2, $3, $4, $5, $6, $7, TRUE, $8, $8)
		ON CONFLICT ("id") DO NOTHING
	`, apikeyID, "seed-key", rawKey[:8], "spr_", keyHash, "default", userID, time.Now())
	if err != nil {
		return "", fmt.Errorf("inserting apikey: %w", err)
	}

	log.Info().Str("apikey_id", apikeyID).Str("key_start", rawKey[:8]+"...").Msg("Seeded API key")

	return rawKey, nil
}
