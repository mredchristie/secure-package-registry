-- Create the jwks table required by BetterAuth's JWT plugin.
-- Stores Ed25519 key pairs used to sign and verify JWTs.
CREATE TABLE IF NOT EXISTS "jwks" (
    "id"         TEXT        NOT NULL PRIMARY KEY,
    "publicKey"  TEXT        NOT NULL,
    "privateKey" TEXT        NOT NULL,
    "createdAt"  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
