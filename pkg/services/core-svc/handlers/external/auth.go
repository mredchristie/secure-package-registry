package external

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strings"

	"git.duti.dev/secure-package-registry/internal/gen/coredb"
	"git.duti.dev/secure-package-registry/pkg/logger"
	"github.com/go-chi/render"
)

// betterAuthCookieName is the session cookie set by BetterAuth in the dashboard.
const betterAuthCookieName = "better-auth.session_token"

// contextKey is an unexported type for context keys in this package.
type contextKey int

const userIDKey contextKey = iota

// UserIDFromContext extracts the authenticated user ID from the request context.
// Returns an empty string if no user ID is present.
func UserIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(userIDKey).(string); ok {
		return v
	}
	return ""
}

// hashAPIKey hashes a raw API key the same way BetterAuth does:
// SHA-256 of the raw key, then base64url-encoded with no padding.
func hashAPIKey(rawKey string) string {
	hash := sha256.Sum256([]byte(rawKey))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// AuthMiddleware returns a chi middleware that authenticates requests via either:
//  1. Bearer API key (CLI usage) — hashed and looked up in the apikey table.
//  2. BetterAuth session cookie (dashboard usage) — token looked up in the session table.
//
// On success the owning user ID is injected into the request context.
func AuthMiddleware(db coredb.Querier) func(http.Handler) http.Handler {
	log := logger.WithComponent("auth-middleware")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Strategy 1: Bearer API key.
			if authHeader := r.Header.Get("Authorization"); authHeader != "" {
				rawToken, ok := strings.CutPrefix(authHeader, "Bearer ")
				if !ok || rawToken == "" {
					render.Status(r, http.StatusUnauthorized)
					render.JSON(w, r, map[string]string{"error": "invalid authorization header"})
					return
				}

				keyHash := hashAPIKey(rawToken)
				userID, err := db.GetAPIKeyOwner(r.Context(), keyHash)
				if err != nil {
					log.Debug().Err(err).Msg("API key lookup failed")
					render.Status(r, http.StatusUnauthorized)
					render.JSON(w, r, map[string]string{"error": "invalid or expired API key"})
					return
				}

				ctx := context.WithValue(r.Context(), userIDKey, userID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Strategy 2: BetterAuth session cookie.
			cookie, err := r.Cookie(betterAuthCookieName)
			if err == nil && cookie.Value != "" {
				userID, err := db.GetSessionUser(r.Context(), cookie.Value)
				if err != nil {
					log.Debug().Err(err).Msg("session cookie lookup failed")
					render.Status(r, http.StatusUnauthorized)
					render.JSON(w, r, map[string]string{"error": "invalid or expired session"})
					return
				}

				ctx := context.WithValue(r.Context(), userIDKey, userID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// No credentials provided.
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, map[string]string{"error": "missing authorization header or session cookie"})
		})
	}
}
