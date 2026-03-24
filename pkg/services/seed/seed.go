// Package seed populates the database with a BetterAuth-compatible user and
// API key for local dev. It is intended to be called once at startup when
// SPR_MOCK=true is set.
package seed

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"git.duti.dev/secure-package-registry/internal/gen/coredb"
	"git.duti.dev/secure-package-registry/pkg/logger"
	"git.duti.dev/secure-package-registry/pkg/services"
)

// Run seeds the database with a dev user and API key, then logs the raw key.
// It is idempotent: conflicting rows are silently skipped.
func Run(ctx context.Context, deps *services.Deps) error {
	log := logger.WithComponent("seed")

	queries := coredb.New(deps.Pool)

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

	// Generate a random 32-byte API key encoded as hex (64 chars).
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		return fmt.Errorf("generating random bytes: %w", err)
	}
	rawKey := hex.EncodeToString(rawBytes)

	// Hash the same way BetterAuth does: SHA-256 -> base64url no-padding.
	hash := sha256.Sum256([]byte(rawKey))
	keyHash := base64.RawURLEncoding.EncodeToString(hash[:])

	// Generate a random ID for the apikey row.
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return fmt.Errorf("generating apikey id: %w", err)
	}
	apikeyID := hex.EncodeToString(idBytes)

	// Insert directly into the apikey table (no sqlc query for insert).
	_, err = deps.Pool.Exec(ctx, `
		INSERT INTO apikey ("id", "name", "start", "prefix", "key", "configId", "referenceId", "enabled", "createdAt", "updatedAt")
		VALUES ($1, $2, $3, $4, $5, $6, $7, TRUE, $8, $8)
		ON CONFLICT ("id") DO NOTHING
	`, apikeyID, "seed-key", rawKey[:8], "spr_", keyHash, "default", userID, time.Now())
	if err != nil {
		return fmt.Errorf("inserting apikey: %w", err)
	}

	log.Info().
		Str("apikey_id", apikeyID).
		Str("key_start", rawKey[:8]+"...").
		Str("raw_key", rawKey).
		Msg("Seeded API key (use as Bearer token)")

	return nil
}
