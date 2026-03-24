-- BetterAuth API key plugin table.
-- Schema matches @better-auth/api-key v1.5.x.
CREATE TABLE "apikey" (
    "id" text NOT NULL PRIMARY KEY,
    "name" text,
    "start" text,
    "prefix" text,
    "key" text NOT NULL,
    "configId" text NOT NULL DEFAULT 'default',
    "referenceId" text NOT NULL,
    "refillInterval" integer,
    "refillAmount" integer,
    "lastRefillAt" timestamptz,
    "enabled" boolean NOT NULL DEFAULT TRUE,
    "rateLimitEnabled" boolean NOT NULL DEFAULT FALSE,
    "rateLimitTimeWindow" integer,
    "rateLimitMax" integer,
    "requestCount" integer NOT NULL DEFAULT 0,
    "remaining" integer,
    "lastRequest" timestamptz,
    "expiresAt" timestamptz,
    "createdAt" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "permissions" text,
    "metadata" text
);

CREATE INDEX "apikey_referenceId_idx" ON "apikey" ("referenceId");
CREATE INDEX "apikey_key_idx" ON "apikey" ("key");
