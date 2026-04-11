// Package seed populates the database with a dev user, project, and project
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

	"git.duti.dev/secure-package-registry/internal/gen/coredb"
	"git.duti.dev/secure-package-registry/pkg/logger"
	"git.duti.dev/secure-package-registry/pkg/services"
)

// Run seeds the database with a dev user, project, and project API key,
// then logs the raw key with npm configuration instructions.
// It is idempotent: conflicting rows are silently skipped.
func Run(ctx context.Context, deps *services.Deps) error {
	log := logger.WithComponent("seed")

	queries := coredb.New(deps.Pool)

	const userID = "seed-user-1"
	const userName = "seed-user"
	const userEmail = "seed@localhost"

	_, err := queries.InsertUser(ctx, coredb.InsertUserParams{
		ID:    userID,
		Name:  userName,
		Email: userEmail,
	})
	if err != nil {
		log.Warn().Err(err).Msg("User may already exist, continuing")
	} else {
		log.Info().Str("user_id", userID).Msg("Seeded user")
	}

	// Create a seed project (or update if it already exists).
	project, err := queries.InsertProject(ctx, coredb.InsertProjectParams{
		UserID:     userID,
		Name:       "seed-project",
		SourceType: "package.json",
		SourceFile: []byte(`{"name":"seed-project","dependencies":{}}`),
	})
	if err != nil {
		return fmt.Errorf("inserting seed project: %w", err)
	}
	log.Info().Int32("project_id", project.ID).Msg("Seeded project")

	// Generate a project API key: "spr_" + 32 random bytes base64url.
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		return fmt.Errorf("generating random bytes: %w", err)
	}
	rawKey := "spr_" + base64.RawURLEncoding.EncodeToString(rawBytes)
	hash := sha256.Sum256([]byte(rawKey))
	keyHash := base64.RawURLEncoding.EncodeToString(hash[:])
	keyPrefix := rawKey[:12]

	// Generate a random ID for the key row.
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return fmt.Errorf("generating key id: %w", err)
	}
	keyID := hex.EncodeToString(idBytes)

	if err := queries.InsertProjectAPIKey(ctx, coredb.InsertProjectAPIKeyParams{
		ID:        keyID,
		ProjectID: project.ID,
		Name:      "seed-key",
		KeyHash:   keyHash,
		Prefix:    keyPrefix,
	}); err != nil {
		return fmt.Errorf("inserting project API key: %w", err)
	}

	proxyURL := deps.Config.ReverseProxy.ExternalURL

	log.Info().
		Str("key_id", keyID).
		Str("prefix", keyPrefix).
		Str("raw_key", rawKey).
		Msg("Seeded project API key")

	log.Info().
		Str("registry", proxyURL+"/npm/").
		Str("token", rawKey).
		Msgf("npm config set //%s/npm/:_authToken %s && npm config set registry %s/npm/",
			stripScheme(proxyURL), rawKey, proxyURL)

	return nil
}

// stripScheme removes the scheme (http:// or https://) from a URL for npm config.
func stripScheme(u string) string {
	for _, prefix := range []string{"https://", "http://"} {
		if after, ok := cutPrefix(u, prefix); ok {
			return after
		}
	}
	return u
}

func cutPrefix(s, prefix string) (string, bool) {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):], true
	}
	return s, false
}
