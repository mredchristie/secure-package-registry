CREATE TABLE organization_packages (
    id SERIAL PRIMARY KEY,
    organization_id text NOT NULL REFERENCES organization(id),
    package_id INTEGER NOT NULL REFERENCES packages(id),
    UNIQUE(organization_id, package_id)
);
