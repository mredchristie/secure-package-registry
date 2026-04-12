// Package regproxy implements the registry reverse proxy service.
// It exposes /npm/* to clients and routes requests to the appropriate Gitea
// account's package registry. Tarball downloads for package versions with a
// "reproducible" tag are served from the registry account (spr-registry);
// all other requests go to the sandbox account (spr-sandbox).
// Requests are gated by per-project API key validation (Bearer token,
// SHA-256/base64url hash lookup) and per-project policy enforcement on
// tarball downloads.
package regproxy

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
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
	"github.com/jackc/pgx/v5"
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

// parseTarballPath extracts the package name and version from an npm tarball
// download path. Returns empty strings if the path is not a tarball request.
//
// Examples:
//
//	/npm/express/-/express-4.18.0.tgz          -> "express", "4.18.0"
//	/npm/@sveltejs%2fkit/-/kit-2.0.0.tgz       -> "@sveltejs/kit", "2.0.0"
//	/npm/@sveltejs/kit/-/kit-2.0.0.tgz         -> "@sveltejs/kit", "2.0.0"
//	/npm/express                                -> "", ""
func parseTarballPath(path string) (pkgName, version string) {
	rest := strings.TrimPrefix(path, "/npm/")
	if rest == path {
		return "", ""
	}

	// Find the /-/ separator that precedes the tarball filename.
	before, after, ok := strings.Cut(rest, "/-/")
	if !ok {
		return "", ""
	}

	rawPkg := before
	tarballFile := after // after "/-/"

	// Decode percent-encoded scoped names: @scope%2fname -> @scope/name
	pkgName, err := url.PathUnescape(rawPkg)
	if err != nil {
		return "", ""
	}

	// The tarball filename is <unscoped-name>-<version>.tgz.
	// For @scope/name, unscoped is "name".
	unscoped := pkgName
	if strings.HasPrefix(pkgName, "@") {
		parts := strings.SplitN(pkgName, "/", 2)
		if len(parts) == 2 {
			unscoped = parts[1]
		}
	}

	// Strip the unscoped name prefix and ".tgz" suffix to get the version.
	prefix := unscoped + "-"
	suffix := ".tgz"
	if !strings.HasPrefix(tarballFile, prefix) || !strings.HasSuffix(tarballFile, suffix) {
		return "", ""
	}
	version = tarballFile[len(prefix) : len(tarballFile)-len(suffix)]

	return pkgName, version
}

// evaluatePolicy checks a package's tags against the project's policy.
// Returns a list of human-readable violations. Empty means allowed.
func evaluatePolicy(project coredb.GetProjectByAPIKeyRow, dep coredb.CheckPackagePolicyRow) []string {
	// If manual review is allowed and the package is manually approved, allow immediately.
	if project.AllowManualReview && dep.ManuallyApproved {
		return nil
	}

	var violations []string

	if project.RequireProvenance && !dep.HasAttestation && !dep.HasOssRebuild {
		violations = append(violations, "provenance check failed: no attestation or oss rebuild")
	}

	if project.RequireBehavior && !dep.BehaviorPassed {
		violations = append(violations, "behavioral analysis check failed")
	}

	return violations
}

// accountForRequest determines which Gitea account should serve the request.
// Tarball downloads for reproducible package versions use the registry account;
// everything else uses the sandbox account.
type accountInfo struct {
	username string
	token    string
	prefix   string // Gitea API path prefix: /api/packages/<username>/npm
}

// Start runs the registry proxy. It blocks until ctx is cancelled, then
// gracefully shuts down.
func Start(ctx context.Context, deps *services.Deps) error {
	cfg := deps.Config.ReverseProxy
	queries := coredb.New(deps.Pool)

	// Read the Gitea config from Valkey to get account credentials.
	giteaConfig, err := deps.Valkey.GetGiteaConfig(ctx)
	if err != nil {
		return fmt.Errorf("loading gitea config from valkey: %w", err)
	}

	sandboxAccount, ok := giteaConfig.Accounts["sandbox"]
	if !ok {
		return fmt.Errorf("gitea config missing sandbox account")
	}
	registryAccount, ok := giteaConfig.Accounts["registry"]
	if !ok {
		return fmt.Errorf("gitea config missing registry account")
	}

	sandbox := accountInfo{
		username: sandboxAccount.Username,
		token:    sandboxAccount.Token,
		prefix:   fmt.Sprintf("/api/packages/%s/npm", sandboxAccount.Username),
	}
	registry := accountInfo{
		username: registryAccount.Username,
		token:    registryAccount.Token,
		prefix:   fmt.Sprintf("/api/packages/%s/npm", registryAccount.Username),
	}

	log.Info().
		Str("gitea_base", giteaConfig.BaseURL).
		Str("sandbox_user", sandbox.username).
		Str("registry_user", registry.username).
		Msg("Loaded Gitea config; routing reproducible builds to registry account")

	target, err := url.Parse(giteaConfig.BaseURL)
	if err != nil {
		return fmt.Errorf("parsing gitea base URL: %w", err)
	}

	// routeKey is stored in request context to tell the director which account to use.
	type routeKey struct{}

	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host

		// Pick the account from context (set by the handler).
		acct := sandbox
		if v, ok := req.Context().Value(routeKey{}).(accountInfo); ok {
			acct = v
		}

		rest := strings.TrimPrefix(req.URL.Path, "/npm")
		req.URL.Path = acct.prefix + rest
		req.Header.Set("Authorization", "Bearer "+acct.token)
	}

	// Rewrite internal Gitea URLs in JSON responses so npm clients see the
	// external proxy URL instead of internal Gitea paths.
	externalURLPrefix := cfg.ExternalURL + "/npm"

	sandboxInternalURL := giteaConfig.BaseURL + sandbox.prefix
	registryInternalURL := giteaConfig.BaseURL + registry.prefix

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

		// Rewrite URLs from both accounts to the external URL.
		rewritten := bytes.ReplaceAll(body, []byte(sandboxInternalURL), []byte(externalURLPrefix))
		rewritten = bytes.ReplaceAll(rewritten, []byte(registryInternalURL), []byte(externalURLPrefix))

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

		// Authenticate via project API key.
		keyHash := hashAPIKey(token)
		project, err := queries.GetProjectByAPIKey(r.Context(), keyHash)
		if err != nil {
			l.Warn().Str("key_hash", keyHash).Msg("Invalid project API key")
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		l = l.With().
			Int32("project_id", project.ID).
			Str("project_name", project.Name).
			Logger()

		l.Debug().Msg("Accepted request")
		r.Header.Del("Authorization")

		// Only GET requests are allowed.
		if r.Method != http.MethodGet {
			l.Warn().Str("method", r.Method).Msg("Blocked non-GET request")
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		// Determine routing: check if this is a tarball download for a
		// reproducible package version.
		acct := sandbox
		pkgName, version := parseTarballPath(r.URL.Path)

		if pkgName != "" && version != "" {
			// --- Policy enforcement on tarball downloads ---
			depCheck, err := queries.CheckPackagePolicy(r.Context(), coredb.CheckPackagePolicyParams{
				ProjectID:  project.ID,
				Ecosystem:  coredb.EcosystemNpm,
				Identifier: pkgName,
				Version:    version,
			})
			if err != nil {
				if err == pgx.ErrNoRows {
					// Package not in project's dependency set — block.
					l.Info().
						Str("package", pkgName).
						Str("version", version).
						Msg("Blocked: package not in project dependency set")
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					_ = json.NewEncoder(w).Encode(map[string]any{
						"error":   "policy violation",
						"package": pkgName,
						"version": version,
						"violations": []string{
							"package not in project dependency set",
						},
					})
					return
				}
				// DB error — fail open with a warning for non-policy queries.
				l.Warn().Err(err).
					Str("package", pkgName).
					Str("version", version).
					Msg("Failed to check package policy, failing open")
			} else {
				// Evaluate policy.
				violations := evaluatePolicy(project, depCheck)
				if len(violations) > 0 {
					l.Info().
						Str("package", pkgName).
						Str("version", version).
						Strs("violations", violations).
						Msg("Blocked by policy")
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					_ = json.NewEncoder(w).Encode(map[string]any{
						"error":      "policy violation",
						"package":    pkgName,
						"version":    version,
						"violations": violations,
					})
					return
				}

				// Check reproducible tag for routing.
				hasRepro, err := queries.HasPackageVersionTag(r.Context(), coredb.HasPackageVersionTagParams{
					Ecosystem:  coredb.EcosystemNpm,
					Identifier: pkgName,
					Version:    version,
					Label:      "reproducible",
				})
				if err != nil {
					l.Warn().Err(err).
						Str("package", pkgName).
						Str("version", version).
						Msg("Failed to check reproducible tag, falling back to sandbox")
				} else if hasRepro {
					acct = registry
					l.Info().
						Str("package", pkgName).
						Str("version", version).
						Msg("Routing to registry (reproducible build)")
				}
			}
		}

		// Store the chosen account in context for the Director.
		r = r.WithContext(context.WithValue(r.Context(), routeKey{}, acct))
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
