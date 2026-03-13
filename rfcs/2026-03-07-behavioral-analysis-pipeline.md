+++
title = "Behavioral Analysis Pipeline"
authors = ["cheongyx@cardiff.ac.uk"]
reviewers = []
creation = "2026-03-07"
depends_on = ["2026-01-30-architectural-design.md", "2026-03-07-npm-dependency-resolution.md"]
+++

## Summary

Defines the behavioral analysis pipeline: how packages flow from version detection through dependency mirroring,
sandboxed execution, and data collection. Supersedes the behavioral analysis runner and anomaly detection sections of
the architectural design RFC.

## Problem

The architectural design RFC described behavioral analysis at a high level: eBPF/ecapture in self-hosted containers with
a separate anomaly detection engine. Through prototyping at HackEurope (see `context-references/hackeurope-spr/`), we
validated the concept but found that the execution environment and orchestration model need to be more concrete.

## Pipeline

Collection is a chained workflow across core-svc and `cmd/be-runner/`:

1. `spr.package.updated` is published for a new package version
2. core-svc consumes the event and inserts a pending collection task in Postgres
3. If the insert succeeds, core-svc publishes `spr.collection.requested`
4. `cmd/be-runner/` consumes `spr.collection.requested`
5. **Resolve dependencies** — Full dependency tree resolution in Go (see `2026-03-07-npm-dependency-resolution.md`)
6. **Mirror to sandbox** — Upload the target package and all transitive dependencies to the Gitea sandbox account
7. **Trigger GitHub Actions workflow** — `workflow_dispatch` with package name, version, and registry preset. Test
   generation happens inside the workflow (see `2026-03-08-github-runner-test-generation.md`), not in `be-runner`.
8. **Poll for completion** — Wait for workflow run to finish. Timeout is caller-controlled via context (currently 10
   minutes). Heartbeats are deferred until the schema and message types exist.
9. **Download artifacts** — The workflow uploads a zip artifact containing `behavior.jsonl`. `be-runner` downloads the
   raw zip via the GitHub API (handling the 302 auth-stripping redirect). If the workflow conclusion is not `success`,
   `be-runner` fails fast and Nacks for retry rather than attempting artifact download.
10. **Store raw data** — Upload artifacts to MinIO _(deferred — not yet implemented)_
11. **Publish result** — Emit a completion event with status, failure details, and artifact locations for core-svc to
    persist _(deferred — not yet implemented)_

## Execution Environment

GitHub Actions (Ubuntu runners) for MVP. The workflow:

- Runs Tracee on the host, scoped to a Docker container PID
- Container resolves packages from the Gitea sandbox registry
- Tests execute sequentially inside the container
- Tracee captures syscalls, file access, network connections, DNS queries
- Artifacts uploaded even if tests crash (malicious packages may intentionally fail)

This is a pragmatic choice. The eBPF/container isolation model from the architectural RFC remains the long-term
direction, but GitHub Actions gives us validated infrastructure now. The service interface (`cmd/be-runner/` consuming
from RabbitMQ, producing collection results) is environment-agnostic — swapping to self-hosted runners or dedicated
nodes is a non-breaking change.

## Gitea Accounts

Two dedicated Gitea accounts, provisioned at infrastructure startup (see `compose.yml`):

- **`spr-registry`** — Curated, approved packages
- **`spr-sandbox`** — Temporary packages for behavioral analysis

Account credentials stored as a single `gitea:config` JSON entry in Valkey. The `pkg/gitea` client provides typed access
to either account's npm registry via role name.

## Task Tracking

core-svc owns task creation and persistence.

- `spr.package.updated` does not directly start `cmd/be-runner/`
- core-svc records a pending collection task first
- Only a successful insert leads to `spr.collection.requested`
- This deduplicates collection requests for the same package version and avoids races between task creation and result
  handling

`cmd/be-runner/` reports progress back through RabbitMQ:

- **Heartbeat events** while polling GitHub Actions
- **Completion event** with terminal status, failure reason, and artifact references

core-svc stores `started_at`, heartbeat timestamps, and final result data. If heartbeats stop for too long, core-svc may
issue a cancellation request and retry. Cancellation handling is not yet designed.

## Open Questions

1. Cancellation flow — message shape, ownership, and runner behavior when cancellation is requested mid-collection
2. AI-based analysis of behavioral data — approach and integration point after collection completes
3. Baseline diffing strategy — how to generate and maintain behavioral baselines across versions
4. Sandbox cleanup — when and how to remove packages from the sandbox account after collection
5. Promotion flow — how a successful collection plus later analysis result triggers upload to the registry account
