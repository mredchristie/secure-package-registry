-- User projects: track dependency sets uploaded by authenticated users.

CREATE TABLE user_projects (
    id          SERIAL PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    source_type TEXT NOT NULL, -- 'package.json' | 'package-lock.json'
    created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (user_id, name)
);

CREATE INDEX idx_user_projects_user_id ON user_projects(user_id);

-- Dependency type: direct (from dependencies/devDependencies) or transitive.
CREATE TYPE DEPENDENCY_TYPE AS ENUM ('direct', 'transitive');

CREATE TABLE project_dependencies (
    id                  SERIAL PRIMARY KEY,
    project_id          INT NOT NULL REFERENCES user_projects(id) ON DELETE CASCADE,
    package_id          INT NOT NULL REFERENCES packages(id),
    package_version_id  INT NOT NULL REFERENCES package_versions(id),
    dependency_type     DEPENDENCY_TYPE NOT NULL,
    version_constraint  TEXT, -- original semver range from package.json (NULL for lock files)
    created_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (project_id, package_id, package_version_id)
);

CREATE INDEX idx_project_deps_project_id ON project_dependencies(project_id);
CREATE INDEX idx_project_deps_package_id ON project_dependencies(package_id);
CREATE INDEX idx_project_deps_version_id ON project_dependencies(package_version_id);
