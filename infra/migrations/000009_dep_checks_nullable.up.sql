-- Add checked_at timestamp to track when checks have been run.
-- When NULL, the dependency checks are pending/not yet completed.
-- When set, the boolean values indicate actual pass/fail status.
ALTER TABLE project_dependencies ADD COLUMN checked_at TIMESTAMPTZ;
