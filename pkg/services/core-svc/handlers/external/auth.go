package external

import (
	"context"
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

// newJWKS creates (or retries) the JWKS keyfunc used by both auth middlewares.
func newJWKS() (keyfunc.Keyfunc, error) {
	return keyfunc.NewDefault([]string{jwksURL})
}

// verifyJWT is shared JWT verification logic. On success it returns the subject
// (user ID) from the token. On failure it writes the appropriate HTTP error
// response and returns an empty string.
func verifyJWT(w http.ResponseWriter, r *http.Request, kf *keyfunc.Keyfunc) string {
	log := logger.WithComponent("auth")

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "missing authorization header"})
		return ""
	}

	rawToken, ok := strings.CutPrefix(authHeader, "Bearer ")
	if !ok || rawToken == "" {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "invalid authorization header"})
		return ""
	}

	// Retry JWKS initialisation if it failed at startup.
	if *kf == nil {
		newKf, err := newJWKS()
		if err != nil {
			log.Error().Err(err).Msg("JWKS keyfunc initialisation failed")
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, map[string]string{"error": "authentication service unavailable"})
			return ""
		}
		*kf = newKf
	}

	parsed, parseErr := jwt.Parse(rawToken, (*kf).Keyfunc,
		jwt.WithValidMethods([]string{"EdDSA"}),
	)
	if parseErr != nil || !parsed.Valid {
		log.Debug().Err(parseErr).Msg("JWT verification failed")
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "invalid or expired token"})
		return ""
	}

	sub, subErr := parsed.Claims.GetSubject()
	if subErr != nil || sub == "" {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "token missing subject claim"})
		return ""
	}

	return sub
}

// AuthMiddleware returns a chi middleware that authenticates dashboard requests
// via Bearer JWT verified against the BetterAuth JWKS endpoint.
// The user ID is extracted from the "sub" claim and injected into the context.
func AuthMiddleware() func(http.Handler) http.Handler {
	log := logger.WithComponent("auth-middleware")

	kf, err := newJWKS()
	if err != nil {
		log.Warn().Err(err).Msg("failed to initialise JWKS keyfunc — JWT auth will retry on first request")
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sub := verifyJWT(w, r, &kf)
			if sub == "" {
				return // verifyJWT already wrote the error response
			}

			ctx := context.WithValue(r.Context(), userIDKey, sub)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AdminAuthMiddleware returns a chi middleware that authenticates requests via
// Bearer JWT (same as AuthMiddleware) and additionally verifies that the
// authenticated user has the "admin" role in the database.
func AdminAuthMiddleware(db coredb.Querier) func(http.Handler) http.Handler {
	log := logger.WithComponent("admin-auth-middleware")

	kf, err := newJWKS()
	if err != nil {
		log.Warn().Err(err).Msg("failed to initialise JWKS keyfunc — JWT auth will retry on first request")
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sub := verifyJWT(w, r, &kf)
			if sub == "" {
				return // verifyJWT already wrote the error response
			}

			// Look up the user's role in the database.
			role, err := db.GetUserRole(r.Context(), sub)
			if err != nil {
				log.Error().Err(err).Str("user_id", sub).Msg("failed to look up user role")
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, map[string]string{"error": "failed to verify admin status"})
				return
			}
			if !role.Valid || role.String != "admin" {
				render.Status(r, http.StatusForbidden)
				render.JSON(w, r, map[string]string{"error": "admin access required"})
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, sub)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
