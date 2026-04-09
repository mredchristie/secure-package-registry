package external

import (
	"context"
	"strings"

	"git.duti.dev/secure-package-registry/pkg/lockfile"
	"git.duti.dev/secure-package-registry/pkg/npm"
	"github.com/rs/zerolog"
)

// ResolveTransitiveDeps resolves the full transitive dependency tree for each
// direct dependency using the npm registry resolver. Returns only the transitive
// deps (direct deps are excluded since they are already in the caller's list).
//
// This is exported so it can be used by both the HTTP handler (for validation)
// and the async consumer (for background processing).
func ResolveTransitiveDeps(ctx context.Context, resolver *npm.Resolver, log zerolog.Logger, directDeps []lockfile.Dep) []lockfile.Dep {
	// Pre-populate seen set with direct deps to avoid duplicates and ensure
	// direct type takes precedence.
	seen := make(map[string]bool, len(directDeps))
	for _, d := range directDeps {
		seen[d.Name+"@"+d.Version] = true
	}

	var transitive []lockfile.Dep

	for _, dep := range directDeps {
		graph, err := resolver.Resolve(ctx, dep.Name, dep.Version)
		if err != nil {
			log.Warn().Err(err).
				Str("dep", dep.Name+"@"+dep.Version).
				Msg("Failed to resolve transitive deps — skipping tree")
			continue
		}

		for key, node := range graph.Nodes {
			if key == graph.Root {
				continue // skip the direct dep itself
			}
			if node == nil {
				continue // incomplete resolution (in-progress placeholder)
			}
			if seen[key] {
				continue
			}
			seen[key] = true

			transitive = append(transitive, lockfile.Dep{
				Name:    node.Name,
				Version: node.Version,
				Direct:  false,
			})
		}
	}

	log.Info().
		Int("direct", len(directDeps)).
		Int("transitive", len(transitive)).
		Msg("Resolved transitive dependency tree")

	return transitive
}

// IsNonStandardSpecifier returns true if the constraint string is a git URL,
// GitHub shorthand, tarball URL, or local file path rather than a semver range.
// These are skipped during resolution (see RFC 2026-04-09).
func IsNonStandardSpecifier(constraint string) bool {
	if constraint == "" {
		return false
	}

	// Git protocols.
	for _, prefix := range []string{"git+", "git://", "git@"} {
		if strings.HasPrefix(constraint, prefix) {
			return true
		}
	}

	// Tarball URLs.
	if strings.HasPrefix(constraint, "http://") || strings.HasPrefix(constraint, "https://") {
		return true
	}

	// Local file paths.
	if strings.HasPrefix(constraint, "file:") {
		return true
	}

	// GitHub shorthand: "user/repo" or "user/repo#ref".
	// Must contain exactly one "/" with no spaces and no semver-range chars.
	if strings.Contains(constraint, "/") &&
		!strings.ContainsAny(constraint, " <>!=^~*|") {
		parts := strings.SplitN(constraint, "#", 2)
		segments := strings.Split(parts[0], "/")
		if len(segments) == 2 && segments[0] != "" && segments[1] != "" {
			return true
		}
	}

	return false
}
