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
    source_url = EXCLUDED.source_url
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
