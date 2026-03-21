+++
title = "Simplified Reproducible Build Integration"
author = ["cheongyx@cardiff.ac.uk"]
reviewers = []
creation = "2026-03-20"
+++

## Summary

We need reproducible builds, but we don't have time to build them "properly" right now. This RFC describes a pragmatic,
throwaway solution that uses Gitea's package registry as an abstraction layer. A standalone build tool produces
artifacts and pushes them to a dedicated Gitea account (`spr-rep`). An internal API validates, registers, and stores
successful builds. Reproducibility status is tracked via package version tags.

When we have more time, we can throw this away and standardize on something like Google's OSS rebuild system.

## Problem

We need to provide reproducible builds for packages in the registry. The previous design involved building a full
microservice with message queues and complex orchestration. This is over-engineered for our current needs and timeline.

## Solution

We chose this architecture because it minimizes integration work while still achieving our goal. By using Gitea's
package registry as the storage layer, we don't need to implement our own artifact storage, authentication, or
distribution mechanism. The build tool is standalone and only needs to make an HTTP call to register results, making it
easy to replace or upgrade later.

### Architecture

1. **Gitea as Storage Layer**: A dedicated Gitea account (`spr-rep`) stores reproducible build artifacts. This is our
   abstraction layer - it doesn't matter how we got the reproducible build, just that we have one.

2. **External Build Tool**: A standalone tool (`cmd/rep-build/`) that:
   - Reads build configurations from `cmd/rep-build/` directory
   - Executes builds in Podman containers
   - Calls the internal API to register successful builds

3. **Internal API**: New endpoint (not public-facing) to register reproducible builds:
   - Validates the package version exists in upstream registry
   - Updates database via tags (reproducibility is just a tag, not a schema change)
   - Forwards the built package to `spr-rep` Gitea account (we already have a Gitea client there)

4. **Reverse Proxy**: Configuration-based routing to `spr-rep` when reproducible builds are available. The database is
   not responsible for routing logic - that's separation of concerns.

### Build Configuration

Build configurations are stored as YAML files in a separate repository (mirrored to GitHub at `cmd/rep-build/` using
`.gitlab-ci.yml` like we do for `spr-gh-runner`). The exact configuration format is to be determined - we need to figure
out what works.

Strategies will include common npm-based build tools (npm, pnpm, yarn) and a "custom" strategy for package-specific
builds when the generic ones don't work.

### Operation

The build tool is triggered manually. We're not building automation yet. On successful completion, it calls the admin
API to register the artifact.

### Build Validation

Diffoscope comparison runs as a separate step outside the reproducible build process. The build process itself just
focuses on building - validation happens elsewhere.

### Scope

We're focusing on npm for now. One reproducible build per version is sufficient - we're not worrying about different
build environments yet.

### Open Questions

1. Should the build tool support batch operations?
2. Do we need build provenance/signing?
