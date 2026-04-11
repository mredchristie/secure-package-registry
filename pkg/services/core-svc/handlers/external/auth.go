package external

import (
	"context"
	"net/http"
	"strings"

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

// AuthMiddleware returns a chi middleware that authenticates dashboard requests
// via Bearer JWT verified against the BetterAuth JWKS endpoint.
// The user ID is extracted from the "sub" claim and injected into the context.
func AuthMiddleware() func(http.Handler) http.Handler {
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

			// JWT from dashboard — verify via JWKS.
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

			ctx := context.WithValue(r.Context(), userIDKey, sub)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
