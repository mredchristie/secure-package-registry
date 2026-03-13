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

-- name: InsertPackage :one
INSERT INTO packages (identifier, ecosystem, latest_version)
VALUES ($1, $2, $3)
ON CONFLICT (identifier) DO UPDATE SET
    latest_version = EXCLUDED.latest_version,
    updated_at = CURRENT_TIMESTAMP
RETURNING id;

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
