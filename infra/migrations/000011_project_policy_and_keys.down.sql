DROP TABLE IF EXISTS project_api_keys;

ALTER TABLE user_projects
    DROP COLUMN IF EXISTS require_provenance,
    DROP COLUMN IF EXISTS require_behavior,
    DROP COLUMN IF EXISTS allow_manual_review;
