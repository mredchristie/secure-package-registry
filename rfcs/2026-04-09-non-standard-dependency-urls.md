+++
title = "Non-Standard Dependency URL Handling"
authors = ["cheongyx@cardiff.ac.uk"]
reviewers = []
creation = "2026-04-09"
depends_on = ["2026-04-01-user-dependency-tracking.md", "2026-03-07-npm-dependency-resolution.md"]
tags = ["feature", "security", "api"]
+++

## Summary

Handle non-standard dependency specifiers (git URLs, GitHub shorthand, tarball URLs) encountered in `package.json`
uploads. These cannot be resolved through the npm registry and represent a distinct security surface. We fetch the
package contents, hash the URL to produce a deterministic identifier under a `@nonstandard` scope, and store the package
in our registry for analysis.

## Problem

When a user uploads a `package.json`, dependencies may use specifiers that are not semver ranges against the npm
registry:

```json
{
  "dependencies": {
    "my-fork": "github:user/repo#abc123",
    "patched-lib": "git+https://github.com/user/repo.git#v2.0.0",
    "vendored": "https://example.com/pkg-1.0.0.tgz",
    "local-link": "file:../lib"
  }
}
```

Today, the upload handler calls `npmClient.ResolveConstraint()` for these, which fails because they are not valid semver
ranges. The dependency is silently skipped (`continue`). This means:

1. **Incomplete dependency tracking** -- the user's project is missing deps from its tracked set.
2. **No security analysis** -- git-sourced packages bypass all behavioral analysis and verification.
3. **Silent data loss** -- no indication to the user that deps were dropped.

Git-sourced dependencies are a known supply chain attack vector. An attacker who compromises a GitHub account can push
malicious code that gets installed directly without going through npm's publish pipeline, provenance checks, or our
behavioral analysis.

## Design

### Detection

A dependency specifier is non-standard if it matches any of:

- `github:user/repo` or `user/repo` (GitHub shorthand)
- `git+https://...`, `git+ssh://...`, `git://...` (git URLs)
- `https://.../*.tgz` or `http://...` (tarball URLs)
- `file:...` (local paths -- cannot be fetched, must be rejected with a warning)

Detection happens after the existing semver constraint resolution fails, or proactively by checking whether the
constraint string matches these patterns before attempting registry resolution.

### Identifier Generation

To store a non-standard package in our system, we need a stable, deterministic identifier:

1. Normalize the URL (strip trailing slashes, lowercase the scheme/host).
2. Compute `SHA-256(normalized_url)` and take the first 16 hex characters.
3. Construct the identifier: `@nonstandard/<hash>` (e.g. `@nonstandard/a1b2c3d4e5f67890`).

The `@nonstandard` scope is reserved and must never conflict with real npm scoped packages. npm scopes are tied to npm
org accounts, and `nonstandard` will not be registered.

### Fetch and Ingest

For git URLs and GitHub shorthand:

1. Clone or fetch the archive from the git host at the specified ref (commit, tag, branch).
2. Extract the `package.json` from the fetched contents.
3. Replace the `name` field with the `@nonstandard/<hash>` identifier.
4. Store the package tarball in our registry (MinIO).
5. Create a `packages` row with the nonstandard identifier and a `package_versions` row.

For tarball URLs:

1. Fetch the tarball directly.
2. Extract `package.json`, replace `name` with the `@nonstandard/<hash>` identifier.
3. Store in our registry.

For `file:` paths:

1. Cannot be fetched remotely. Log a warning and skip.
2. Include the skipped dependency in the upload response so the user is informed.

### Parent Dependency Rewriting

When the non-standard dependency is a **direct** dependency of the user's project, we store the
`project_dependencies` row with the `@nonstandard/<hash>` identifier and the original URL in the
`version_constraint` column for traceability.

When it appears as a transitive dependency (resolved from a direct dep's own dependency tree), the resolver already
cannot follow it. We skip it and log a warning -- transitive non-standard deps are uncommon and usually indicate vendored
forks.

### Metadata Tracking

Add a `source_url` column usage: the `package_versions` table already has a `source_url` column. For nonstandard
packages, populate it with the original URL so the provenance is traceable:

```text
package.identifier     = "@nonstandard/a1b2c3d4e5f67890"
package_version.version = "0.0.0-git.abc123"  (derived from ref)
package_version.source_url = "git+https://github.com/user/repo.git#abc123"
```

### Version Derivation

Since non-standard deps have no semver version, we derive a synthetic one:

- Git ref is a tag matching semver (e.g. `v2.0.0`): use the semver (`2.0.0`).
- Git ref is a commit hash: `0.0.0-git.<short-hash>`.
- Git ref is a branch name: `0.0.0-branch.<branch-name>`.
- Tarball URL: `0.0.0-url.<first-8-of-sha256-of-url>`.

### Security Analysis

Non-standard packages should be treated with **higher suspicion** than registry packages:

- Always trigger behavioral analysis regardless of direct/transitive status.
- Skip verification (attestation + OSS rebuild) since these packages have no npm provenance.
- Flag them in the project summary as `source: nonstandard` so the user sees them called out.

## Decisions

1. **`@nonstandard` scope.** Scoped under a reserved namespace to prevent collisions with real packages. The hash is
   deterministic so the same URL always maps to the same identifier.
2. **`file:` paths are skipped.** We cannot fetch local files. These are rejected with a clear warning in the response.
3. **Tarball and git URLs are fetched server-side.** This means core-svc needs outbound HTTP/git access, which it already
   has for npm registry calls.
4. **Original URL preserved.** Stored in `version_constraint` (for project deps) and `source_url` (for package versions)
   for full traceability.
5. **Higher security scrutiny.** Non-registry packages bypass npm's publish pipeline and provenance, so they always get
   behavioral analysis.

## Implementation Phases

### Phase 1 (Current): Skip and Warn

Non-standard specifiers are detected and skipped with a log warning. The upload response includes a count of skipped
dependencies so the user is informed. No silent data loss.

### Phase 2 (Future): Fetch and Store

Implement the full fetch, hash, rewrite, and store pipeline described above. Requires:

- URL pattern detection utility in `pkg/lockfile` or `pkg/npm`.
- Git archive fetching (via `go-git` or GitHub archive API).
- Tarball download and extraction.
- Integration with MinIO storage.

### Phase 3 (Future): Enhanced Analysis

- Diff non-standard packages against their upstream npm counterparts (if they are forks).
- Track ref changes across re-uploads (did the git ref change? was it force-pushed?).
- Alert users when a non-standard dep's source repository is compromised or deleted.

## Open Questions

1. **Rate limiting fetches** -- Should we limit how many non-standard URLs we fetch per upload to prevent abuse?
2. **Caching** -- Should we cache fetched tarballs by URL+ref, or always re-fetch on upload?
3. **Private repositories** -- Should we support authenticated git clones (user provides a token)? This adds significant
   complexity and credential management concerns.
