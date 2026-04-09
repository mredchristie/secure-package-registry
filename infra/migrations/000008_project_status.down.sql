ALTER TABLE user_projects
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS source_file,
    DROP COLUMN IF EXISTS generation;
