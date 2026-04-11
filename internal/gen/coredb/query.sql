-- name: GetPackageVersion :one
SELECT
    p.identifier,
    p.ecosystem::text,
    pv.version,
    (pv.version = p.latest_version) AS latest,
    pv.source_url,
    pv.source_tag,
    pv.source_commit_hash,
    p.maintainer_trust_level AS trust_level,
    pv.maintainer_notes
FROM packages p
JOIN package_versions pv ON pv.package_id = p.id
WHERE p.ecosystem = $1
  AND p.identifier = $2
  AND pv.version = $3;

-- name: GetPackageVersionTags :many
SELECT
    ptt.label,
    ptt.value_type,
    pvt.value
FROM packages p
JOIN package_versions pv ON pv.package_id = p.id
JOIN package_version_tags pvt ON pvt.package_version = pv.id
JOIN package_tag_types ptt ON ptt.id = pvt.tag_type
WHERE p.ecosystem = $1
  AND p.identifier = $2
  AND pv.version = $3;

-- name: SearchPackages :many
SELECT
    p.identifier,
    p.ecosystem::text,
    p.latest_version
FROM packages p
WHERE p.identifier ILIKE '%' || sqlc.arg(query) || '%'
  AND (sqlc.narg(ecosystem)::ECOSYSTEM IS NULL OR p.ecosystem = sqlc.narg(ecosystem)::ECOSYSTEM)
ORDER BY p.identifier
LIMIT sqlc.arg(page_size) OFFSET (sqlc.arg(page) - 1) * sqlc.arg(page_size);

-- name: CountSearchPackages :one
SELECT COUNT(*)
FROM packages p
WHERE p.identifier ILIKE '%' || sqlc.arg(query) || '%'
  AND (sqlc.narg(ecosystem)::ECOSYSTEM IS NULL OR p.ecosystem = sqlc.narg(ecosystem)::ECOSYSTEM);

-- name: InsertPackage :one
INSERT INTO packages (identifier, ecosystem, latest_version)
VALUES ($1, $2, $3)
ON CONFLICT (identifier) DO UPDATE SET
    latest_version = COALESCE(EXCLUDED.latest_version, packages.latest_version),
    updated_at = CURRENT_TIMESTAMP
RETURNING id, (xmax = 0) AS inserted;

-- name: InsertPackageVersion :one
INSERT INTO package_versions (package_id, version, source_url)
VALUES ($1, $2, $3)
ON CONFLICT (package_id, version) DO UPDATE SET
    source_url = COALESCE(EXCLUDED.source_url, package_versions.source_url),
    updated_at = CURRENT_TIMESTAMP
RETURNING id;

-- name: InsertTagType :one
INSERT INTO package_tag_types (label, description, value_type)
VALUES ($1, $2, $3)
ON CONFLICT (label) DO UPDATE SET
    description = EXCLUDED.description,
    value_type = EXCLUDED.value_type
RETURNING id;

-- name: InsertPackageTag :exec
INSERT INTO package_version_tags (package_version, tag_type, value)
VALUES ($1, $2, $3)
ON CONFLICT (package_version, tag_type) DO UPDATE SET
    value = EXCLUDED.value;

-- Poller queries

-- name: ListPackagesByEcosystem :many
SELECT id, identifier, ecosystem, latest_version
FROM packages
WHERE ecosystem = $1;

-- name: ListPackageVersions :many
SELECT version
FROM package_versions
WHERE package_id = $1;

-- name: UpdatePackageLatestVersion :exec
UPDATE packages
SET latest_version = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: GetPackageByEcosystemAndIdentifier :one
SELECT id, identifier, ecosystem, latest_version
FROM packages
WHERE ecosystem = $1
  AND identifier = $2;

-- Collection task queries

-- name: InsertCollectionTask :one
-- Inserts a new collection task for a package version + source.
-- Returns the new row. If a task already exists for this combination,
-- does nothing and returns nothing (caller checks sql.ErrNoRows).
INSERT INTO collection_tasks (package_version_id, source, status)
VALUES ($1, $2, 'pending')
ON CONFLICT (package_version_id, source) DO NOTHING
RETURNING id, package_version_id, source, status, created_at;

-- name: HasActiveCollectionTask :one
-- Checks whether an active (pending or running) collection task exists
-- for the given package version and source. Returns true/false.
SELECT EXISTS(
    SELECT 1 FROM collection_tasks
    WHERE package_version_id = $1
      AND source = $2
      AND status IN ('pending', 'running')
) AS active;

-- name: UpdateCollectionTaskStatus :exec
UPDATE collection_tasks
SET status = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: UpdateCollectionTaskRunning :exec
UPDATE collection_tasks
SET status = 'running',
    workflow_run_id = $2,
    started_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: UpdateCollectionTaskHeartbeat :exec
UPDATE collection_tasks
SET heartbeat_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: UpdateCollectionTaskSucceeded :exec
UPDATE collection_tasks
SET status = 'succeeded',
    artifact_bucket = $2,
    artifact_key = $3,
    completed_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: UpdateCollectionTaskFailed :exec
UPDATE collection_tasks
SET status = 'failed',
    failure_reason = $2,
    completed_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: ResetCollectionTask :exec
-- Resets a failed/cancelled task back to pending for retry.
UPDATE collection_tasks
SET status = 'pending',
    workflow_run_id = NULL,
    artifact_bucket = NULL,
    artifact_key = NULL,
    started_at = NULL,
    heartbeat_at = NULL,
    completed_at = NULL,
    failure_reason = NULL,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
  AND status IN ('failed', 'cancelled');

-- name: GetCollectionTask :one
SELECT id, package_version_id, source, status,
       workflow_run_id, artifact_bucket, artifact_key,
       started_at, heartbeat_at, completed_at, failure_reason,
       created_at, updated_at
FROM collection_tasks
WHERE id = $1;

-- name: ListCollectionTasks :many
-- Lists collection tasks with package context, optionally filtered by ecosystem.
-- Ordered by most recently created first, paginated.
SELECT
    ct.id,
    ct.source,
    ct.status,
    ct.artifact_bucket,
    ct.artifact_key,
    ct.failure_reason,
    ct.started_at,
    ct.completed_at,
    ct.created_at,
    p.identifier,
    p.ecosystem::text,
    pv.version
FROM collection_tasks ct
JOIN package_versions pv ON pv.id = ct.package_version_id
JOIN packages p ON p.id = pv.package_id
WHERE (sqlc.narg(ecosystem)::ECOSYSTEM IS NULL OR p.ecosystem = sqlc.narg(ecosystem)::ECOSYSTEM)
ORDER BY ct.created_at DESC
LIMIT sqlc.arg(page_size) OFFSET (sqlc.arg(page) - 1) * sqlc.arg(page_size);

-- name: GetSucceededCollectionTask :one
-- Finds the succeeded collection task for a given ecosystem, package identifier, and version.
-- Returns the artifact location needed for serving deduped behavior data.
SELECT
    ct.id,
    ct.artifact_bucket,
    ct.artifact_key
FROM collection_tasks ct
JOIN package_versions pv ON pv.id = ct.package_version_id
JOIN packages p ON p.id = pv.package_id
WHERE p.ecosystem = $1
  AND p.identifier = $2
  AND pv.version = $3
  AND ct.status = 'succeeded'
  AND ct.artifact_key IS NOT NULL
LIMIT 1;

-- Tag queries

-- name: GetTagTypeByLabel :one
-- Looks up a tag type ID by its label.
SELECT id FROM package_tag_types WHERE label = $1;

-- name: GetPackageVersionID :one
-- Looks up a package version row ID by ecosystem, identifier, and version.
SELECT pv.id
FROM package_versions pv
JOIN packages p ON p.id = pv.package_id
WHERE p.ecosystem = $1
  AND p.identifier = $2
  AND pv.version = $3;

-- name: HasPackageVersionTag :one
-- Checks whether a package version has a specific tag (by label) set to a truthy value.
SELECT EXISTS(
    SELECT 1
    FROM package_version_tags pvt
    JOIN package_tag_types ptt ON ptt.id = pvt.tag_type
    JOIN package_versions pv ON pv.id = pvt.package_version
    JOIN packages p ON p.id = pv.package_id
    WHERE p.ecosystem = $1
      AND p.identifier = $2
      AND pv.version = $3
      AND ptt.label = $4
      AND pvt.value = 'true'::jsonb
) AS has_tag;

-- Auth queries

-- name: GetAPIKeyOwner :one
SELECT "referenceId" FROM apikey
WHERE key = $1
  AND enabled = TRUE
  AND ("expiresAt" IS NULL OR "expiresAt" > NOW());

-- name: GetSessionUser :one
-- Looks up a BetterAuth session token and returns the owning user ID,
-- provided the session has not expired.
SELECT "userId" FROM "session"
WHERE "token" = $1
  AND "expiresAt" > NOW();

-- name: GetUserRole :one
-- Returns the role for a given user ID.
SELECT "role" FROM "user"
WHERE "id" = $1;

-- User/Organisation Queries

-- name: InsertUser :one
INSERT INTO "user" (
    "id",
    "name",
    "email",
    "emailVerified",
    "createdAt",
    "updatedAt"
) VALUES ($1, $2, $3, FALSE, NOW(), NOW())
ON CONFLICT (id) DO NOTHING
RETURNING "id";

-- Project queries

-- name: InsertProject :one
-- Creates or updates a user project. On conflict (same user+name), updates
-- the source_type, stores the raw file for async processing, resets status
-- to pending, and bumps the generation counter.
INSERT INTO user_projects (user_id, name, source_type, status, source_file, generation)
VALUES ($1, $2, $3, 'pending', $4, 1)
ON CONFLICT (user_id, name) DO UPDATE SET
    source_type = EXCLUDED.source_type,
    status      = 'pending',
    source_file = EXCLUDED.source_file,
    generation  = user_projects.generation + 1,
    updated_at  = CURRENT_TIMESTAMP
RETURNING id, user_id, name, source_type, status, source_file, generation, require_provenance, require_behavior, allow_manual_review, created_at, updated_at;

-- name: GetProject :one
SELECT id, user_id, name, source_type, status, generation, require_provenance, require_behavior, allow_manual_review, created_at, updated_at
FROM user_projects
WHERE id = $1;

-- name: GetProjectByUserAndName :one
SELECT id, user_id, name, source_type, status, generation, require_provenance, require_behavior, allow_manual_review, created_at, updated_at
FROM user_projects
WHERE user_id = $1 AND name = $2;

-- name: ListUserProjects :many
SELECT id, user_id, name, source_type, status, generation, require_provenance, require_behavior, allow_manual_review, created_at, updated_at
FROM user_projects
WHERE user_id = $1
ORDER BY updated_at DESC;

-- name: GetProjectForProcessing :one
-- Fetches the project with its raw source file for background processing.
-- Used by the consumer to retrieve the file to parse and resolve.
SELECT id, user_id, name, source_type, status, source_file, generation, require_provenance, require_behavior, allow_manual_review, created_at, updated_at
FROM user_projects
WHERE id = $1;

-- name: FinishProjectProcessing :exec
-- Marks a project as complete or failed after async processing, clears the
-- raw source_file, and only applies if the generation matches (stale-message guard).
UPDATE user_projects
SET status      = $2,
    source_file = NULL,
    updated_at  = CURRENT_TIMESTAMP
WHERE id = $1
  AND generation = $3;

-- name: DeleteProject :exec
DELETE FROM user_projects
WHERE id = $1 AND user_id = $2;

-- name: DeleteProjectDependencies :exec
-- Bulk delete all dependencies for a project (used before re-inserting on re-upload).
DELETE FROM project_dependencies
WHERE project_id = $1;

-- name: InsertProjectDependency :exec
-- Inserts a single project dependency. ON CONFLICT ignores duplicates.
INSERT INTO project_dependencies (project_id, package_id, package_version_id, dependency_type, version_constraint)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (project_id, package_id, package_version_id) DO NOTHING;

-- name: ListProjectDependencies :many
-- Lists all dependencies for a project with package info and per-dep check statuses.
-- Optionally filtered by dependency type.
-- Check fields are NULL when checked_at is NULL (not yet checked), otherwise boolean.
SELECT
    pd.id,
    pd.dependency_type,
    pd.version_constraint,
    p.identifier,
    p.ecosystem::text,
    pv.version,
    pd.package_version_id,
    pd.package_id,
    (CASE WHEN att.value IS NOT NULL THEN att.value = 'true'::jsonb ELSE NULL END) AS has_attestation,
    (CASE WHEN oss.value IS NOT NULL THEN oss.value = 'true'::jsonb ELSE NULL END) AS has_oss_rebuild,
    (CASE WHEN beh.value IS NOT NULL THEN beh.value = 'true'::jsonb ELSE NULL END) AS behavior_passed
FROM project_dependencies pd
JOIN packages p ON p.id = pd.package_id
JOIN package_versions pv ON pv.id = pd.package_version_id
LEFT JOIN package_version_tags att
    ON att.package_version = pd.package_version_id
    AND att.tag_type = (SELECT id FROM package_tag_types WHERE label = 'upstream_attestation')
LEFT JOIN package_version_tags oss
    ON oss.package_version = pd.package_version_id
    AND oss.tag_type = (SELECT id FROM package_tag_types WHERE label = 'oss_rebuild')
LEFT JOIN package_version_tags beh
    ON beh.package_version = pd.package_version_id
    AND beh.tag_type = (SELECT id FROM package_tag_types WHERE label = 'behavior_passed')
WHERE pd.project_id = $1
  AND (sqlc.narg(dep_type)::DEPENDENCY_TYPE IS NULL OR pd.dependency_type = sqlc.narg(dep_type)::DEPENDENCY_TYPE)
ORDER BY pd.dependency_type, p.identifier;

-- name: GetProjectSummary :many
-- Aggregated stats for a project, grouped by dependency type.
-- Returns total count plus counts of deps with each boolean tag set to true.
SELECT
    pd.dependency_type,
    COUNT(*)::int AS total,
    COUNT(*) FILTER (WHERE att.value = 'true'::jsonb)::int AS has_attestation,
    COUNT(*) FILTER (WHERE oss.value = 'true'::jsonb)::int AS has_oss_rebuild,
    COUNT(*) FILTER (WHERE beh.value = 'true'::jsonb)::int AS behavior_passed
FROM project_dependencies pd
LEFT JOIN package_version_tags att
    ON att.package_version = pd.package_version_id
    AND att.tag_type = (SELECT id FROM package_tag_types WHERE label = 'upstream_attestation')
LEFT JOIN package_version_tags oss
    ON oss.package_version = pd.package_version_id
    AND oss.tag_type = (SELECT id FROM package_tag_types WHERE label = 'oss_rebuild')
LEFT JOIN package_version_tags beh
    ON beh.package_version = pd.package_version_id
    AND beh.tag_type = (SELECT id FROM package_tag_types WHERE label = 'behavior_passed')
WHERE pd.project_id = $1
GROUP BY pd.dependency_type;

-- name: GetPackageDependents :many
-- Inverse query: find which projects depend on a given package.
-- Used for impact analysis ("who is affected if this package is compromised?").
SELECT
    up.id AS project_id,
    up.user_id,
    up.name AS project_name,
    pd.dependency_type,
    pv.version
FROM project_dependencies pd
JOIN user_projects up ON up.id = pd.project_id
JOIN package_versions pv ON pv.id = pd.package_version_id
WHERE pd.package_id = $1
ORDER BY up.user_id, up.name;

-- name: ListPackageVersionsForReview :many
-- Lists package versions that failed behavioral analysis, with their review status.
-- Used by the admin review queue. Optionally filtered by ecosystem.
-- Sorted: unreviewed first (NULL manually_approved), then by identifier.
SELECT
    p.identifier,
    p.ecosystem::text,
    pv.version,
    (pv.version = p.latest_version) AS is_latest,
    (SELECT pvt.value
     FROM package_version_tags pvt
     JOIN package_tag_types ptt ON ptt.id = pvt.tag_type
     WHERE pvt.package_version = pv.id AND ptt.label = 'manually_approved') AS manually_approved,
    (SELECT pvt.value
     FROM package_version_tags pvt
     JOIN package_tag_types ptt ON ptt.id = pvt.tag_type
     WHERE pvt.package_version = pv.id AND ptt.label = 'review_comment') AS review_comment
FROM package_versions pv
JOIN packages p ON p.id = pv.package_id
JOIN package_version_tags beh ON beh.package_version = pv.id
    AND beh.tag_type = (SELECT id FROM package_tag_types WHERE label = 'behavior_passed')
    AND beh.value = 'false'::jsonb
WHERE (sqlc.narg(ecosystem)::ECOSYSTEM IS NULL OR p.ecosystem = sqlc.narg(ecosystem)::ECOSYSTEM)
ORDER BY
    (SELECT pvt.value FROM package_version_tags pvt
     JOIN package_tag_types ptt ON ptt.id = pvt.tag_type
     WHERE pvt.package_version = pv.id AND ptt.label = 'manually_approved') IS NULL DESC,
    p.identifier, pv.version;

-- name: GetVersionReviewStatus :one
-- Gets the manual review status and comment for a specific package version.
SELECT
    (SELECT pvt.value
     FROM package_version_tags pvt
     JOIN package_tag_types ptt ON ptt.id = pvt.tag_type
     WHERE pvt.package_version = pv.id AND ptt.label = 'manually_approved') AS manually_approved,
    (SELECT pvt.value
     FROM package_version_tags pvt
     JOIN package_tag_types ptt ON ptt.id = pvt.tag_type
     WHERE pvt.package_version = pv.id AND ptt.label = 'review_comment') AS review_comment
FROM package_versions pv
JOIN packages p ON p.id = pv.package_id
WHERE p.ecosystem = $1
  AND p.identifier = $2
  AND pv.version = $3;

-- Project policy queries

-- name: GetProjectPolicy :one
SELECT id, require_provenance, require_behavior, allow_manual_review
FROM user_projects
WHERE id = $1;

-- name: UpdateProjectPolicy :exec
UPDATE user_projects
SET require_provenance  = $2,
    require_behavior    = $3,
    allow_manual_review = $4,
    updated_at          = CURRENT_TIMESTAMP
WHERE id = $1;

-- Project API key queries

-- name: InsertProjectAPIKey :exec
INSERT INTO project_api_keys (id, project_id, name, key_hash, prefix, expires_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListProjectAPIKeys :many
SELECT id, project_id, name, prefix, expires_at, created_at
FROM project_api_keys
WHERE project_id = $1
ORDER BY created_at DESC;

-- name: DeleteProjectAPIKey :exec
DELETE FROM project_api_keys
WHERE id = $1 AND project_id = $2;

-- name: GetProjectByAPIKey :one
-- Looks up a project by hashed API key. Returns the project with policy fields.
SELECT
    up.id,
    up.user_id,
    up.name,
    up.require_provenance,
    up.require_behavior,
    up.allow_manual_review
FROM project_api_keys pak
JOIN user_projects up ON up.id = pak.project_id
WHERE pak.key_hash = $1
  AND (pak.expires_at IS NULL OR pak.expires_at > NOW());

-- name: CheckPackagePolicy :one
-- Given a project and a package (by ecosystem+identifier+version), checks whether
-- the package is in the project's dependency set and returns its tag values.
-- Returns sql.ErrNoRows if the package is not in the dep set (→ block).
SELECT
    pd.id AS dep_id,
    COALESCE(att.value = 'true'::jsonb, false)::bool AS has_attestation,
    COALESCE(oss.value = 'true'::jsonb, false)::bool AS has_oss_rebuild,
    COALESCE(beh.value = 'true'::jsonb, false)::bool AS behavior_passed,
    COALESCE(man.value = 'true'::jsonb, false)::bool AS manually_approved
FROM project_dependencies pd
JOIN packages p ON p.id = pd.package_id
JOIN package_versions pv ON pv.id = pd.package_version_id
LEFT JOIN package_version_tags att
    ON att.package_version = pd.package_version_id
    AND att.tag_type = (SELECT id FROM package_tag_types WHERE label = 'upstream_attestation')
LEFT JOIN package_version_tags oss
    ON oss.package_version = pd.package_version_id
    AND oss.tag_type = (SELECT id FROM package_tag_types WHERE label = 'oss_rebuild')
LEFT JOIN package_version_tags beh
    ON beh.package_version = pd.package_version_id
    AND beh.tag_type = (SELECT id FROM package_tag_types WHERE label = 'behavior_passed')
LEFT JOIN package_version_tags man
    ON man.package_version = pd.package_version_id
    AND man.tag_type = (SELECT id FROM package_tag_types WHERE label = 'manually_approved')
WHERE pd.project_id = $1
  AND p.ecosystem = $2
  AND p.identifier = $3
  AND pv.version = $4
LIMIT 1;
-- name: ListPackageVersionsPublic :many
-- Lists all versions for a package by ecosystem+identifier, with their verification tags.
SELECT
    pv.version,
    (pv.version = p.latest_version) AS latest,
    pv.source_url,
    pv.source_tag,
    pv.source_commit_hash,
    COALESCE((SELECT pvt.value = 'true'
     FROM package_version_tags pvt
     JOIN package_tag_types ptt ON ptt.id = pvt.tag_type
     WHERE pvt.package_version = pv.id AND ptt.label = 'upstream_attestation'), false)::bool AS has_attestation,
    COALESCE((SELECT pvt.value = 'true'
     FROM package_version_tags pvt
     JOIN package_tag_types ptt ON ptt.id = pvt.tag_type
     WHERE pvt.package_version = pv.id AND ptt.label = 'oss_rebuild'), false)::bool AS has_oss_rebuild,
    (SELECT pvt.value
     FROM package_version_tags pvt
     JOIN package_tag_types ptt ON ptt.id = pvt.tag_type
     WHERE pvt.package_version = pv.id AND ptt.label = 'behavior_passed') AS behavior_passed,
    (SELECT pvt.value
     FROM package_version_tags pvt
     JOIN package_tag_types ptt ON ptt.id = pvt.tag_type
     WHERE pvt.package_version = pv.id AND ptt.label = 'manually_approved') AS manually_approved,
    (SELECT pvt.value
     FROM package_version_tags pvt
     JOIN package_tag_types ptt ON ptt.id = pvt.tag_type
     WHERE pvt.package_version = pv.id AND ptt.label = 'review_comment') AS review_comment
FROM package_versions pv
JOIN packages p ON p.id = pv.package_id
WHERE p.ecosystem = $1
  AND p.identifier = $2
ORDER BY pv.created_at DESC;
