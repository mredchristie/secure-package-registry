CREATE TYPE ECOSYSTEM AS ENUM ('npm', 'go', 'cargo', 'pypi');

CREATE TABLE packages (
    id SERIAL PRIMARY KEY,
    identifier TEXT UNIQUE NOT NULL,
    ecosystem ECOSYSTEM NOT NULL,
    latest_version TEXT NOT NULL,
    maintainer_trust_level INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_packages_lookup ON packages (identifier, ecosystem);

CREATE TABLE package_versions (
    id SERIAL PRIMARY KEY,
    package_id INTEGER NOT NULL REFERENCES packages(id) ON DELETE CASCADE,
    version TEXT NOT NULL,
    source_url TEXT NOT NULL,
    source_tag TEXT,
    source_commit_hash TEXT,
    maintainer_notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(package_id, version)
);

CREATE INDEX idx_versions_lookup ON package_versions (package_id, version);

CREATE TYPE PKG_VTYPE AS ENUM ('integer', 'boolean', 'float');

CREATE TABLE package_tag_types (
    id SERIAL PRIMARY KEY,
    label TEXT NOT NULL,
    description TEXT,
    value_type PKG_VTYPE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE package_version_tags (
    package_version INTEGER NOT NULL REFERENCES package_versions(id) ON DELETE CASCADE,
    tag_type INTEGER NOT NULL REFERENCES package_tag_types(id) ON DELETE CASCADE,
    value JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (package_version, tag_type)
);
