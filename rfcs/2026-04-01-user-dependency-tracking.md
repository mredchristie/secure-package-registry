+++
title = "User Dependency Tracking"
authors = ["cheongyx@cardiff.ac.uk"]
reviewers = []
creation = "2026-04-01"
depends_on = ["2026-03-07-behavioral-analysis-pipeline.md", "2026-03-07-npm-dependency-resolution.md", "2026-03-24-registry-authentication-flow.md", "2026-03-07-core-data-task-collection.md"]
tags = ["feature", "schema", "api", "dashboard"]
+++

## Summary

Allow authenticated users to upload `package.json` or lock files (`package-lock.json`) to create **projects** whose
dependency trees are tracked, analyzed, and monitored by SPR. The system distinguishes between **direct** and
**transitive** dependencies, automatically triggers behavioral analysis for direct dependencies, runs verification
(upstream attestation + OSS rebuild) for all dependencies, and surfaces project-level and per-dependency aggregated
security posture.

## Problem

Today, SPR tracks packages that are manually added by admins or discovered by the npm poller. There is no concept of a
user's actual dependency set. Users cannot answer:

- "Are any of my dependencies behaving anomalously?"
- "How many of my dependencies have provenance attestations?"
- "What is the security posture of my project's supply chain?"

Without knowing _who depends on what_, we also cannot answer the inverse: "Which users are affected if package X is
compromised?"

## Design

### Core Concepts

**Project** -- A named container representing a user's application. A user can have multiple projects. Each project has a
single active dependency set derived from the most recent upload.

**Project Dependency** -- A record linking a project to a specific package at a specific version, annotated as either
`direct` or `transitive`.

- **Direct dependencies** come from the top-level `dependencies` and `devDependencies` fields. `devDependencies` are
  treated as direct because they execute during `npm install` and `npm test`, making them a real attack surface.
- **Transitive dependencies** are everything else in the resolved tree.

### Upload Flow

1. User uploads a `package.json` or lock file via the dashboard (or CLI).
2. The backend parses the file, extracts direct dependencies.
3. For `package.json`: semver ranges are resolved to latest matching versions using the existing Go-native npm resolver
   (`pkg/npm/resolve.go`). The full transitive tree is resolved.
4. For `package-lock.json` (v2/v3): exact versions and the full dependency tree are extracted directly from the lock
   structure. Direct dependencies are identified from the root entry's `dependencies`/`devDependencies`. If the lock file
   format does not clearly distinguish direct from transitive, we identify packages with no parents in the dependency
   graph.
5. The project's dependency set is replaced atomically (delete old + insert new within a transaction).
6. For each **direct** dependency not already being analyzed: the package is created in our system, and behavioral
   analysis + verification are triggered automatically. Limited to **5 concurrent behavioral analysis runs per user** to
   prevent resource exhaustion.
7. For each **transitive** dependency: only verification (attestation + OSS rebuild) is triggered. Transitive deps do not
   need independent behavioral analysis.

### Direct vs. Transitive: Behavioral Analysis Scope

The behavioral analysis pipeline (`be-runner`) already resolves and installs the full transitive tree when testing a
package. When we sandbox `express@4.18.2`, we also install and exercise all of express's transitive dependencies. If a
transitive dep does something malicious during install, it will be caught in the parent's behavioral trace.

Therefore, transitive dependencies get **transitively tested** through their parent's pipeline run. Only direct
dependencies need their own dedicated behavioral analysis collection task. This dramatically reduces the number of
sandbox runs needed.

### Database Schema

New migration `000006_user_projects`:

```sql
CREATE TABLE user_projects (
    id          SERIAL PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    source_type TEXT NOT NULL, -- 'package.json' | 'package-lock.json'
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, name)
);

CREATE TYPE DEPENDENCY_TYPE AS ENUM ('direct', 'transitive');

CREATE TABLE project_dependencies (
    id                  SERIAL PRIMARY KEY,
    project_id          INT NOT NULL REFERENCES user_projects(id) ON DELETE CASCADE,
    package_id          INT NOT NULL REFERENCES packages(id),
    package_version_id  INT NOT NULL REFERENCES package_versions(id),
    dependency_type     DEPENDENCY_TYPE NOT NULL,
    version_constraint  TEXT, -- original semver range from package.json (NULL for lock files)
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (project_id, package_id, package_version_id)
);
```

The flat, deduplicated schema stores each dependency once per project regardless of how many paths lead to it. Parent
relationships can be reconstructed from the resolver output when needed for per-dependency drill-down views.

### API Endpoints

New endpoints on the external API, all authenticated and user-scoped:

| Method   | Path                          | Description                                      |
| -------- | ----------------------------- | ------------------------------------------------ |
| `POST`   | `/projects`                   | Create project by uploading a file (multipart)   |
| `GET`    | `/projects`                   | List the authenticated user's projects           |
| `GET`    | `/projects/{id}`              | Project detail with dependency summary           |
| `PUT`    | `/projects/{id}`              | Re-upload to update the dependency set           |
| `DELETE` | `/projects/{id}`              | Delete a project                                 |
| `GET`    | `/projects/{id}/dependencies` | List dependencies with type, version, tag status |
| `GET`    | `/projects/{id}/summary`      | Aggregated stats per dependency type             |

Scoped packages (e.g., `@types/node`, `@babel/core`) require URL encoding of the `@` and `/` characters.

### File Parsing

**`package.json`:**

- Extract `name` for the project name.
- Extract `dependencies` and `devDependencies` as direct deps (semver ranges).
- Resolve each to a concrete version using `pkg/npm` resolver.
- Build full transitive tree using `Resolver.Resolve()` for each direct dep.
- Deduplicate across all trees.

**`package-lock.json` (v2/v3):**

- Extract `name` for the project name.
- The `packages` field contains all resolved dependencies with exact versions.
- The root entry's `dependencies`/`devDependencies` identifies direct deps.
- Everything else is transitive.
- No resolution needed -- versions are already pinned.

### Aggregation

**Per-project summary:**

```sql
SELECT
    pd.dependency_type,
    COUNT(*) AS total,
    COUNT(*) FILTER (WHERE att.value = 'true') AS has_attestation,
    COUNT(*) FILTER (WHERE oss.value = 'true') AS has_oss_rebuild,
    COUNT(*) FILTER (WHERE beh.value = 'true') AS behavior_passed
FROM project_dependencies pd
LEFT JOIN package_version_tags att ...
LEFT JOIN package_version_tags oss ...
LEFT JOIN package_version_tags beh ...
WHERE pd.project_id = $1
GROUP BY pd.dependency_type;
```

**Per-direct-dependency aggregation:** For a specific direct dependency, count how many of its transitive deps have each
tag. This reconstructs the tree from resolver output rather than storing parent relationships in the DB.

### Authentication

All project endpoints require authentication. The user ID is passed from the dashboard to core-svc via an `X-User-ID`
header on internal API calls. The dashboard-to-core-svc boundary is internal (same compose network), and the dashboard
has already authenticated the user via BetterAuth.

### Concurrency Limits

Uploading a large lock file could trigger many behavioral analysis runs. To prevent resource exhaustion:

- **5 concurrent behavioral analysis runs per user.** The upload handler tracks how many collection tasks it creates and
  stops triggering new ones after the limit. Remaining direct deps still get verification checks.
- Future enhancement: queue excess runs and process them as slots free up.

### CLI

A CLI tool (`cmd/spr-cli/`) replicates the core upload functionality for testing and developer workflows:

```shell
spr-cli project upload --file package-lock.json --name my-project
spr-cli project list
spr-cli project show <id>
spr-cli project dependencies <id>
spr-cli project summary <id>
spr-cli project delete <id>
```

The CLI authenticates via BetterAuth API keys (same mechanism as `reg-proxy`).

## Decisions

1. **DevDependencies are direct.** They execute during install and test, making them a real attack vector.
2. **Flat dependency storage.** No `parent_package_id` in the schema. Parent relationships are reconstructed from
   resolver output when needed. This avoids fan-out and keeps the table clean.
3. **Re-upload replaces.** Uploading a new file for an existing project atomically replaces the dependency set within a
   transaction. No snapshot history for now.
4. **5 concurrent runs per user.** Prevents a single large upload from monopolizing the behavioral analysis pipeline.
5. **Transitive deps skip behavioral analysis.** They are exercised through their parent's sandbox run. Only verification
   (attestation + OSS rebuild) is triggered for transitive deps.

## Open Questions

1. **Scope handling** -- Scoped packages like `@types/node` need URL encoding throughout the API and routing layers.
2. **Rate limiting** -- Beyond the 5-concurrent-run limit, should there be a cooldown between uploads?
3. **Lock file without package.json** -- If a `package-lock.json` does not clearly identify direct dependencies (unlikely
   in v2/v3 format but possible), we fall back to identifying packages with no parents in the dependency graph.
