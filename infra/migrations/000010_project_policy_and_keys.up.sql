-- Policy columns on user_projects: enforce checks before package downloads.
-- Defaults are strict: require provenance + behavior, no manual override.
ALTER TABLE user_projects
    ADD COLUMN require_provenance  BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN require_behavior    BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN allow_manual_review BOOLEAN NOT NULL DEFAULT FALSE;

-- Per-project API keys for reg-proxy authentication.
CREATE TABLE project_api_keys (
    id          TEXT        NOT NULL PRIMARY KEY,
    project_id  INT         NOT NULL REFERENCES user_projects(id) ON DELETE CASCADE,
    name        TEXT        NOT NULL,
    key_hash    TEXT        NOT NULL UNIQUE,
    prefix      TEXT        NOT NULL,
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_project_api_keys_project ON project_api_keys(project_id);
CREATE INDEX idx_project_api_keys_hash    ON project_api_keys(key_hash);
