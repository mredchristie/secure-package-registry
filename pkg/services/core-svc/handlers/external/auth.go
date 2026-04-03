package external

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strings"

	"git.duti.dev/secure-package-registry/internal/gen/coredb"
	"git.duti.dev/secure-package-registry/pkg/logger"
	"github.com/MicahParks/keyfunc/v3"
	"github.com/go-chi/render"
	"github.com/golang-jwt/jwt/v5"
)

// jwksURL is the BetterAuth JWKS endpoint inside the compose network.
const jwksURL = "http://dashboard-ui:3000/api/auth/jwks"

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

// isJWT returns true if the token looks like a JWT (base64url-encoded JSON
// header starting with "{").
func isJWT(token string) bool {
	return strings.HasPrefix(token, "eyJ")
}

// AuthMiddleware returns a chi middleware that authenticates requests via either:
//  1. Bearer JWT (dashboard) — verified against the BetterAuth JWKS endpoint,
//     user ID extracted from the "sub" claim.
//  2. Bearer API key (CLI) — SHA-256 hashed and looked up in the apikey table.
//
// On success the owning user ID is injected into the request context.
func AuthMiddleware(db coredb.Querier) func(http.Handler) http.Handler {
	log := logger.WithComponent("auth-middleware")

	// Initialise a JWKS keyfunc that fetches and caches public keys from the
	// BetterAuth JWKS endpoint. It automatically re-fetches when it encounters
	// an unknown kid (handles key rotation).
	kf, err := keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		log.Warn().Err(err).Msg("failed to initialise JWKS keyfunc — JWT auth will retry on first request")
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, map[string]string{"error": "missing authorization header"})
				return
			}

			rawToken, ok := strings.CutPrefix(authHeader, "Bearer ")
			if !ok || rawToken == "" {
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, map[string]string{"error": "invalid authorization header"})
				return
			}

			var userID string

			if isJWT(rawToken) {
				// Strategy 1: JWT from dashboard — verify via JWKS.
				if kf == nil {
					// Retry initialisation if it failed at startup.
					kf, err = keyfunc.NewDefault([]string{jwksURL})
					if err != nil {
						log.Error().Err(err).Msg("JWKS keyfunc initialisation failed")
						render.Status(r, http.StatusInternalServerError)
						render.JSON(w, r, map[string]string{"error": "authentication service unavailable"})
						return
					}
				}

				parsed, parseErr := jwt.Parse(rawToken, kf.Keyfunc,
					jwt.WithValidMethods([]string{"EdDSA"}),
				)
				if parseErr != nil || !parsed.Valid {
					log.Debug().Err(parseErr).Msg("JWT verification failed")
					render.Status(r, http.StatusUnauthorized)
					render.JSON(w, r, map[string]string{"error": "invalid or expired token"})
					return
				}

				sub, subErr := parsed.Claims.GetSubject()
				if subErr != nil || sub == "" {
					render.Status(r, http.StatusUnauthorized)
					render.JSON(w, r, map[string]string{"error": "token missing subject claim"})
					return
				}
				userID = sub
			} else {
				// Strategy 2: API key from CLI — hash and look up.
				keyHash := hashAPIKey(rawToken)
				var dbErr error
				userID, dbErr = db.GetAPIKeyOwner(r.Context(), keyHash)
				if dbErr != nil {
					log.Debug().Err(dbErr).Msg("API key lookup failed")
					render.Status(r, http.StatusUnauthorized)
					render.JSON(w, r, map[string]string{"error": "invalid or expired API key"})
					return
				}
			}

			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
