CREATE TABLE organisations (
    id SERIAL PRIMARY KEY,
    name TEXT UNIQUE NOT NULL
);

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    organisation_id INTEGER REFERENCES organisations(id)
);

CREATE TABLE organisation_packages (
    id SERIAL PRIMARY KEY,
    organisation_id INTEGER NOT NULL REFERENCES organisations(id),
    package_id INTEGER NOT NULL REFERENCES packages(id),
    UNIQUE(organisation_id, package_id)
);
