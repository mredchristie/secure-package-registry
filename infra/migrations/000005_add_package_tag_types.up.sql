-- Seed well-known tag types used for package version classification.
-- These are reference data, not schema changes.

INSERT INTO package_tag_types (label, description, value_type) VALUES
    ('reproducible',      'Package version has a verified reproducible build',  'boolean'),
    ('behavior_passed',   'Package version passed behavioral analysis',         'boolean'),
    ('upstream_attestation', 'Package version has attestation from upstream registry', 'boolean'),
    ('oss_rebuild',       'Package version verified via OSS rebuild',           'boolean'),
    ('manually_approved', 'Package version has been manually approved',         'boolean')
ON CONFLICT (label) DO UPDATE SET
    description = EXCLUDED.description,
    value_type  = EXCLUDED.value_type;
