-- Add async processing columns to user_projects.
-- status: tracks whether the background dependency resolution is pending/complete/failed.
-- source_file: stores the raw uploaded file while processing; NULLed after completion.
-- generation: monotonically increasing counter to detect stale re-delivery after re-upload.

ALTER TABLE user_projects
    ADD COLUMN status      TEXT  NOT NULL DEFAULT 'pending',
    ADD COLUMN source_file BYTEA,
    ADD COLUMN generation  INT   NOT NULL DEFAULT 1;
