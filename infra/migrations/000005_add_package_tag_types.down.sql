DELETE FROM package_version_tags
WHERE tag_type IN (
    SELECT id FROM package_tag_types
WHERE label IN ('reproducible', 'behavior_passed', 'upstream_attestation', 'oss_rebuild', 'manually_approved')
);

DELETE FROM package_tag_types
WHERE label IN ('reproducible', 'behavior_passed', 'npm_attestation', 'oss_rebuild', 'manually_approved');
