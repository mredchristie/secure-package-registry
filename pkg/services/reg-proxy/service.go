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

// ---------------------------------------------------------------------------
// Request types
// ---------------------------------------------------------------------------

// npmRoute classifies the kind of npm registry request.
type npmRoute int

const (
	// npmRouteUnknown is a path we don't recognise or refuse to serve.
	npmRouteUnknown npmRoute = iota
	// npmRoutePackument is a full package metadata request: GET /npm/<pkg>
	npmRoutePackument
	// npmRouteTarball is a tarball download: GET /npm/<pkg>/-/<name>-<ver>.tgz
	npmRouteTarball
)

func (r npmRoute) String() string {
	switch r {
	case npmRoutePackument:
		return "packument"
	case npmRouteTarball:
		return "tarball"
	default:
		return "unknown"
	}
}

// npmRequest holds parsed information about an npm registry request path.
type npmRequest struct {
	Route   npmRoute
	Package string // full package name, e.g. "express" or "@sveltejs/kit"
	Version string // version string; only set for tarball requests
}

// parseNpmPath classifies the incoming path under /npm/ into a concrete route.
//
// Recognised patterns:
//
//	/npm/<pkg>                        → packument  (unscoped)
//	/npm/@<scope>%2f<name>            → packument  (scoped, percent-encoded)
//	/npm/@<scope>/<name>              → packument  (scoped, slash form)
//	/npm/<pkg>/-/<unscoped>-<ver>.tgz → tarball    (unscoped)
//	/npm/@<scope>%2f<name>/-/…        → tarball    (scoped)
//	/npm/@<scope>/<name>/-/…          → tarball    (scoped)
//
// Anything else returns npmRouteUnknown.
func parseNpmPath(path string) npmRequest {
	rest := strings.TrimPrefix(path, "/npm/")
	if rest == path || rest == "" {
		return npmRequest{Route: npmRouteUnknown}
	}

	// --- Tarball: contains /-/ separator ---
	if before, after, ok := strings.Cut(rest, "/-/"); ok {
		return parseTarball(before, after)
	}

	// --- Packument: everything else under /npm/<pkg> ---
	return parsePackument(rest)
}

// parseTarball extracts the package name and version from the two halves
// around the /-/ separator.
//
// Gitea uses a non-standard tarball path that includes a version segment:
//
//	standard npm: <pkg>/-/<unscoped>-<ver>.tgz
//	Gitea:        <pkg>/-/<ver>/<unscoped>-<ver>.tgz
//
// Both forms are accepted.
func parseTarball(rawPkg, afterSep string) npmRequest {
	unknown := npmRequest{Route: npmRouteUnknown}

	pkgName, err := url.PathUnescape(rawPkg)
	if err != nil {
		return unknown
	}

	// Determine the unscoped portion of the name (for @scope/name → name).
	unscoped := pkgName
	if strings.HasPrefix(pkgName, "@") {
		parts := strings.SplitN(pkgName, "/", 2)
		if len(parts) != 2 || parts[1] == "" {
			return unknown
		}
		unscoped = parts[1]
	}

	// Strip an optional leading version segment (Gitea format):
	//   "7.19.2/undici-types-7.19.2.tgz" → "undici-types-7.19.2.tgz"
	tarballFile := afterSep
	if i := strings.LastIndex(afterSep, "/"); i != -1 {
		tarballFile = afterSep[i+1:]
	}

	// Tarball filename must be <unscoped>-<version>.tgz.
	prefix := unscoped + "-"
	suffix := ".tgz"
	if !strings.HasPrefix(tarballFile, prefix) || !strings.HasSuffix(tarballFile, suffix) {
		return unknown
	}
	version := tarballFile[len(prefix) : len(tarballFile)-len(suffix)]
	if version == "" {
		return unknown
	}

	return npmRequest{Route: npmRouteTarball, Package: pkgName, Version: version}
}

// parsePackument validates and decodes the package name portion of a packument
// request.
func parsePackument(raw string) npmRequest {
	unknown := npmRequest{Route: npmRouteUnknown}

	decoded, err := url.PathUnescape(raw)
	if err != nil {
		return unknown
	}

	// Scoped: must be exactly @scope/name (one slash after @).
	if strings.HasPrefix(decoded, "@") {
		parts := strings.SplitN(decoded, "/", 3)
		// Exactly 2 parts: @scope and name. Reject if more segments exist.
		if len(parts) != 2 || parts[0] == "@" || parts[1] == "" {
			return unknown
		}
		return npmRequest{Route: npmRoutePackument, Package: decoded}
	}

	// Unscoped: must be a single segment (no slashes).
	if strings.Contains(decoded, "/") {
		return unknown
	}

	return npmRequest{Route: npmRoutePackument, Package: decoded}
}

// ---------------------------------------------------------------------------
// Response types
// ---------------------------------------------------------------------------

// policyViolationResponse is returned when a request is blocked by policy.
type policyViolationResponse struct {
	Error      string   `json:"error"`
	Package    string   `json:"package"`
	Version    string   `json:"version,omitempty"`
	Violations []string `json:"violations"`
}

// writePolicyBlock writes a JSON 403 response for a policy violation.
func writePolicyBlock(w http.ResponseWriter, pkg, version string, violations []string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(policyViolationResponse{
		Error:      "policy violation",
		Package:    pkg,
		Version:    version,
		Violations: violations,
	})
}

// ---------------------------------------------------------------------------
// Auth & policy helpers
// ---------------------------------------------------------------------------

// hashAPIKey hashes a raw API key the same way BetterAuth does:
// SHA-256 of the raw key, then base64url-encoded with no padding.
func hashAPIKey(rawKey string) string {
	hash := sha256.Sum256([]byte(rawKey))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// evaluatePolicy checks a package's tags against the project's policy.
// Returns a list of human-readable violations. Empty means allowed.
func evaluatePolicy(project coredb.GetProjectByAPIKeyRow, dep coredb.CheckPackagePolicyRow) []string {
	// If manual review is allowed and the package is manually approved, allow immediately.
	if project.AllowManualReview && dep.ManuallyApproved {
		return nil
	}

	var violations []string

	if project.RequireProvenance && !dep.HasAttestation && !dep.HasOssRebuild && !dep.HasReproducible {
		violations = append(violations, "provenance check failed: no attestation, oss rebuild, or reproducible build")
	}

	if project.RequireBehavior {
		if dep.DependencyType == coredb.DependencyTypeDirect {
			if !dep.BehaviorAnalyzed || !dep.BehaviorPassed {
				violations = append(violations, "behavioral analysis check failed")
			}
		}
	}

	return violations
}

// ---------------------------------------------------------------------------
// Gitea account routing
// ---------------------------------------------------------------------------

// accountInfo holds credentials and the Gitea API path prefix for one account.
type accountInfo struct {
	username string
	token    string
	prefix   string // e.g. /api/packages/<username>/npm
}

// routeKey is the context key used to pass the chosen account to the Director.
type routeKey struct{}

// ---------------------------------------------------------------------------
// Start
// ---------------------------------------------------------------------------

// Start runs the registry proxy. It blocks until ctx is cancelled, then
// gracefully shuts down.
func Start(ctx context.Context, deps *services.Deps) error {
	cfg := deps.Config.ReverseProxy
	queries := coredb.New(deps.Pool)

	// ---- Gitea config ----
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

	// ---- Reverse proxy ----
	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)

			acct := sandbox
			if v, ok := pr.In.Context().Value(routeKey{}).(accountInfo); ok {
				acct = v
			}

			rest := strings.TrimPrefix(pr.Out.URL.Path, "/npm")
			pr.Out.URL.Path = acct.prefix + rest
			pr.Out.Header.Set("Authorization", "Bearer "+acct.token)
		},
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

		rewritten := bytes.ReplaceAll(body, []byte(sandboxInternalURL), []byte(externalURLPrefix))
		rewritten = bytes.ReplaceAll(rewritten, []byte(registryInternalURL), []byte(externalURLPrefix))

		resp.Body = io.NopCloser(bytes.NewBuffer(rewritten))
		resp.ContentLength = int64(len(rewritten))
		resp.Header.Set("Content-Length", strconv.Itoa(len(rewritten)))
		return nil
	}

	// ---- HTTP handler ----
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := log.With().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Logger()

		// Only GET requests are allowed.
		if r.Method != http.MethodGet {
			l.Warn().Msg("Blocked non-GET request")
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse the route.
		req := parseNpmPath(r.URL.Path)
		l = l.With().
			Str("route", req.Route.String()).
			Str("package", req.Package).
			Str("version", req.Version).
			Logger()

		if req.Route == npmRouteUnknown {
			l.Debug().Msg("Rejected unrecognised path")
			http.NotFound(w, r)
			return
		}

		// Authenticate via project API key.
		auth := r.Header.Get("Authorization")
		token, hasBearer := strings.CutPrefix(auth, "Bearer ")
		if !hasBearer {
			l.Warn().Msg("Missing or non-Bearer auth header")
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

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
			Bool("require_provenance", project.RequireProvenance).
			Bool("require_behavior", project.RequireBehavior).
			Bool("allow_manual_review", project.AllowManualReview).
			Logger()

		l.Info().Msg("Authenticated request")
		r.Header.Del("Authorization")

		// Dispatch by route type.
		switch req.Route {
		case npmRoutePackument:
			handlePackument(w, r, l, proxy, req, project, queries, sandbox)
		case npmRouteTarball:
			handleTarball(w, r, l, proxy, req, project, queries, sandbox, registry)
		}
	})

	// ---- Server lifecycle ----
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

// ---------------------------------------------------------------------------
// Route handlers
// ---------------------------------------------------------------------------

// handlePackument serves package metadata (packument) requests.
// No version-specific policy is enforced here; the request is proxied to the
// sandbox account.
func handlePackument(
	w http.ResponseWriter,
	r *http.Request,
	l zerolog.Logger,
	proxy *httputil.ReverseProxy,
	req npmRequest,
	_ coredb.GetProjectByAPIKeyRow,
	_ *coredb.Queries,
	sandbox accountInfo,
) {
	l.Info().Msg("Proxying packument request")
	r = r.WithContext(context.WithValue(r.Context(), routeKey{}, sandbox))
	proxy.ServeHTTP(w, r)
}

// handleTarball serves tarball download requests with full policy enforcement.
func handleTarball(
	w http.ResponseWriter,
	r *http.Request,
	l zerolog.Logger,
	proxy *httputil.ReverseProxy,
	req npmRequest,
	project coredb.GetProjectByAPIKeyRow,
	queries *coredb.Queries,
	sandbox, registry accountInfo,
) {
	depCheck, err := queries.CheckPackagePolicy(r.Context(), coredb.CheckPackagePolicyParams{
		ProjectID:  project.ID,
		Ecosystem:  coredb.EcosystemNpm,
		Identifier: req.Package,
		Version:    req.Version,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			l.Info().Msg("Blocked tarball: package not in project dependency set")
			writePolicyBlock(w, req.Package, req.Version, []string{
				"package not in project dependency set",
			})
			return
		}
		l.Error().Err(err).Msg("Database error checking package policy")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	l.Info().
		Bool("has_attestation", depCheck.HasAttestation).
		Bool("has_oss_rebuild", depCheck.HasOssRebuild).
		Bool("has_reproducible", depCheck.HasReproducible).
		Bool("behavior_analyzed", depCheck.BehaviorAnalyzed).
		Bool("behavior_passed", depCheck.BehaviorPassed).
		Bool("manually_approved", depCheck.ManuallyApproved).
		Str("dependency_type", string(depCheck.DependencyType)).
		Msg("Policy check result")

	violations := evaluatePolicy(project, depCheck)
	if len(violations) > 0 {
		l.Warn().
			Strs("violations", violations).
			Msg("Blocked tarball: policy violation")
		writePolicyBlock(w, req.Package, req.Version, violations)
		return
	}

	// Determine routing: check if this version has a reproducible build.
	acct := sandbox
	hasRepro, err := queries.HasPackageVersionTag(r.Context(), coredb.HasPackageVersionTagParams{
		Ecosystem:  coredb.EcosystemNpm,
		Identifier: req.Package,
		Version:    req.Version,
		Label:      "reproducible",
	})
	if err != nil {
		l.Warn().Err(err).Msg("Failed to check reproducible tag, falling back to sandbox")
	} else if hasRepro {
		acct = registry
		l.Info().Msg("Routing tarball to registry (reproducible build)")
	}

	l.Info().Str("account", acct.username).Msg("Proxying tarball request")
	r = r.WithContext(context.WithValue(r.Context(), routeKey{}, acct))
	proxy.ServeHTTP(w, r)
}
