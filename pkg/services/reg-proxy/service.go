// Package regproxy implements the registry reverse proxy service.
// It exposes /npm/* to clients and routes requests to the appropriate Gitea
// account's package registry. Currently all reads go to the sandbox account;
// future work will route based on package tags (e.g. reproducibility status).
// Requests are gated by BetterAuth API key validation (Bearer token,
// SHA-256/base64url hash lookup).
package regproxy

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"git.duti.dev/secure-package-registry/internal/gen/coredb"
	"git.duti.dev/secure-package-registry/pkg/httpserver"
	"git.duti.dev/secure-package-registry/pkg/logger"
	"git.duti.dev/secure-package-registry/pkg/services"
	"github.com/rs/zerolog"
)

var log zerolog.Logger

func init() {
	log = logger.WithComponent("reg-proxy")
}

// hashAPIKey hashes a raw API key the same way BetterAuth does:
// SHA-256 of the raw key, then base64url-encoded with no padding.
func hashAPIKey(rawKey string) string {
	hash := sha256.Sum256([]byte(rawKey))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// Start runs the registry proxy. It blocks until ctx is cancelled, then
// gracefully shuts down.
func Start(ctx context.Context, deps *services.Deps) error {
	cfg := deps.Config.ReverseProxy
	queries := coredb.New(deps.Pool)

	// Read the Gitea config from Valkey to get the sandbox account credentials.
	giteaConfig, err := deps.Valkey.GetGiteaConfig(ctx)
	if err != nil {
		return fmt.Errorf("loading gitea config from valkey: %w", err)
	}

	sandboxAccount, ok := giteaConfig.Accounts["sandbox"]
	if !ok {
		return fmt.Errorf("gitea config missing sandbox account")
	}

	log.Info().
		Str("gitea_base", giteaConfig.BaseURL).
		Str("sandbox_user", sandboxAccount.Username).
		Msg("Loaded Gitea config; routing reads to sandbox account")

	target, err := url.Parse(giteaConfig.BaseURL)
	if err != nil {
		return fmt.Errorf("parsing gitea base URL: %w", err)
	}

	// The internal Gitea path prefix for the sandbox account's npm registry.
	internalPrefix := fmt.Sprintf("/api/packages/%s/npm", sandboxAccount.Username)

	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host

		// Rewrite /npm/<rest> -> /api/packages/<sandbox_user>/npm/<rest>
		rest := strings.TrimPrefix(req.URL.Path, "/npm")
		req.URL.Path = internalPrefix + rest

		// Inject the sandbox account's Gitea token so Gitea authorises the
		// request. The client's BetterAuth Bearer token has already been
		// validated and stripped at this point.
		req.Header.Set("Authorization", "Bearer "+sandboxAccount.Token)
	}

	// Rewrite internal Gitea URLs in JSON responses so npm clients see the
	// external proxy URL instead of internal Gitea paths.
	// e.g. http://gitea:3000/api/packages/spr-sandbox/npm/... -> http://localhost:7002/npm/...
	internalURLPrefix := giteaConfig.BaseURL + internalPrefix
	externalURLPrefix := cfg.ExternalURL + "/npm"

	proxy.ModifyResponse = func(resp *http.Response) error {
		ct := resp.Header.Get("Content-Type")
		if !strings.Contains(ct, "application/json") {
			return nil
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if err := resp.Body.Close(); err != nil {
			return err
		}

		rewritten := bytes.ReplaceAll(
			body,
			[]byte(internalURLPrefix),
			[]byte(externalURLPrefix),
		)

		resp.Body = io.NopCloser(bytes.NewBuffer(rewritten))
		resp.ContentLength = int64(len(rewritten))
		resp.Header.Set("Content-Length", strconv.Itoa(len(rewritten)))
		return nil
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := log.With().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Logger()

		// Only /npm/* paths are served; everything else is not found.
		if !strings.HasPrefix(r.URL.Path, "/npm/") && r.URL.Path != "/npm" {
			http.NotFound(w, r)
			return
		}

		auth := r.Header.Get("Authorization")
		token, hasBearer := strings.CutPrefix(auth, "Bearer ")

		if !hasBearer {
			l.Debug().Msg("Missing or non-Bearer auth header")
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		keyHash := hashAPIKey(token)
		ownerID, err := queries.GetAPIKeyOwner(r.Context(), keyHash)
		if err != nil {
			l.Warn().Str("key_hash", keyHash).Msg("Invalid API key")
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		l.Debug().Str("owner", ownerID).Msg("Accepted request")
		r.Header.Del("Authorization")

		// Only GET requests are allowed. npm publish/publish/unpublish require
		// different permissions and are not yet supported.
		if r.Method != http.MethodGet {
			l.Warn().Str("method", r.Method).Msg("Blocked non-GET request")
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		proxy.ServeHTTP(w, r)
	})

	srv := httpserver.New("reg-proxy", "0.0.0.0:7002", handler)
	errCh := srv.Start()

	select {
	case <-ctx.Done():
		log.Info().Msg("Shutting down reg-proxy...")
	case err := <-errCh:
		return err
	}

	shutdownCtx := context.Background()
	if err := srv.Stop(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Failed to stop reg-proxy server")
		return err
	}

	log.Info().Msg("Shutdown complete")
	return nil
}
