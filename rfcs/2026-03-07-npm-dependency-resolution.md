+++
title = "NPM Dependency Resolution"
authors = ["cheongyx@cardiff.ac.uk"]
reviewers = []
creation = "2026-03-07"
+++

## Summary

A Go-native NPM dependency resolver in `pkg/npm/` that builds a full transitive dependency tree without shelling out to
`npm`. Required by the behavioral analysis pipeline to mirror packages to the Gitea sandbox registry.

## Problem

The behavioral analysis pipeline needs the complete dependency graph of a package before running tests. The sandbox
container resolves packages from the Gitea registry, meaning all transitive dependencies must be uploaded beforehand.

The HackEurope prototype solved this by shelling out to `npm`. This is undesirable:

- Requires `npm` installed in the service container
- No control over resolution behavior, retry logic, or parallelism
- Output parsing is fragile

## Recommendation

Implement resolution against the NPM registry package endpoint (`GET /{package}`), which returns all versions and their
dependency maps. The resolver:

1. Fetches the package for the root package
2. Selects the version matching the requested range
3. Extracts `dependencies`, `optionalDependencies`, and `peerDependencies`
4. Recursively resolves each dependency in parallel
5. Produces a flat graph of `(name, version, tarball URL)` tuples with no duplicates

Key concerns:

- **Semver resolution** — Must correctly resolve ranges (`^1.2.3`, `~1.2.3`, `>=1.0.0 <2.0.0`, etc.) against available
  versions. Use an existing Go semver library.
- **Parallelism** — Dependency trees can be large (hundreds of packages). Fetches should be concurrent with bounded
  parallelism and rate limiting to avoid hammering the registry.
- **Cycle detection** — Circular dependencies exist in NPM. The resolver must track visited `(name, version)` pairs.
- **Deduplication** — Multiple packages may depend on the same transitive dependency. Resolve each unique
  `(name, version)` pair only once.

## Open Questions

1. Peer dependency handling — NPM's peer dependency resolution is complex and version-dependent. For MVP, we may treat
   unmet peer dependencies as warnings rather than errors.
2. `optionalDependencies` — Should we resolve these? They may fail to install legitimately. Including them gives broader
   behavioral coverage but adds noise.
3. Lock file support — Should we support `package-lock.json` as an input for exact resolution? Not needed for the
   initial use case (resolving from a name + version) but useful later.
