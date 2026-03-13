+++
title = "Core Data Schema Update for Behavioral Analysis"
authors = ["cheongyx@cardiff.ac.uk"]
reviewers = []
creation = "2026-03-07"
last_updated = "2026-03-08"
supersedes = ["2026-02-04-core-data-schema.md"]
depends_on = ["2026-03-07-behavioral-analysis-pipeline.md"]
+++

## Summary

Extends the core data schema to track collection tasks, link raw artifacts in MinIO, and record later analysis results
per package version.

## Problem

The existing schema tracks packages and versions with tag-based metadata (`behavior_clean`, `anomaly_score`). This is
insufficient for the collection pipeline, which produces artifacts that need to be referenced and collection tasks that
need to be tracked over time.

## Recommendation

The schema needs to support:

- **Collection tasks** — Which package version is being collected, when it was requested, when it started, workflow run
  ID, latest heartbeat, terminal status, and failure reason
- **Artifact references** — MinIO bucket and object keys for `behavior.jsonl` and any future collection outputs
- **Analysis results** — A later step linked to collected artifacts, stored separately from collection task execution

Raw behavioral data lives in MinIO. Postgres tracks metadata and references only.

core-svc should insert the collection task before publishing `spr.collection.requested`. This row is the deduplication
point for package version collection.

The schema should reserve space for heartbeat-driven recovery:

- `started_at` when be-runner begins work
- heartbeat timestamps while GitHub Actions is polled
- terminal status details from the completion event

Cancellation and retry policy are driven by these timestamps, but the exact cancellation design remains open.

## Decisions

### 1. Tags coexist with collection tasks

The existing tag system (`behavior_clean`, `anomaly_score`, etc.) coexists with collection tasks and future analysis
records. Tags remain the user-facing output; collection tasks and analysis results are internal pipeline state. No
replacement is planned.

### 2. MinIO bucket structure

Single bucket with prefixed keys: `behavior/{ecosystem}/{package}/{version}/{source}/behavior.jsonl`. The `source`
dimension handles different collection origins (npm, reproducible build, GitHub releases, etc.). Currently only `npm` is
implemented.

### 3. Collection task status model

Five-state enum (`COLLECTION_TASK_STATUS`): `pending`, `running`, `succeeded`, `failed`, `cancelled`. Tasks start as
`pending` when core-svc inserts them. `be-runner` transitions to `running` when work begins, then to `succeeded` or
`failed` on completion. `cancelled` is reserved for timeout-based cancellation (not yet implemented).

### 4. Retries update the existing row

Failed or cancelled collection tasks are retried by resetting the same row back to `pending` via `ResetCollectionTask`,
which clears all execution state (workflow run ID, timestamps, artifact references, failure reason). No separate attempt
records are created.

### 5. `source_url` made nullable

`package_versions.source_url` was changed from `NOT NULL` to nullable. When core-svc receives `spr.package.updated`, it
only has ecosystem, identifier, and version — no source URL. The `InsertPackageVersion` upsert uses
`COALESCE(EXCLUDED.source_url, package_versions.source_url)` so that inserting with a null `source_url` does not
overwrite an existing non-null value.

### 6. Deduplication via unique constraint

`collection_tasks` has a `UNIQUE (package_version_id, source)` constraint. `InsertCollectionTask` uses
`ON CONFLICT DO NOTHING RETURNING` — when a conflict occurs, no row is returned and the caller checks for
`pgx.ErrNoRows` to detect the duplicate. core-svc also pre-checks with `HasActiveCollectionTask` (pending or running
status) before attempting the insert, providing a fast path that avoids unnecessary conflict handling.

## Open Questions

1. Should the existing tag system be replaced by or coexist with structured collection and analysis records?
   **Resolved** — coexist (see Decisions §1)
2. MinIO bucket structure — single bucket with prefixed keys, or per-package buckets? **Resolved** — single bucket with
   prefixed keys (see Decisions §2)
3. How should collection task status be modeled before the later analysis stage is designed? **Resolved** — five-state
   enum (see Decisions §3)
4. Should retries update the same collection row or create a separate attempt record? **Resolved** — same row (see
   Decisions §4)
