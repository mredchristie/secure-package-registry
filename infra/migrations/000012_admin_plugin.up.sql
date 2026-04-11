-- Better Auth admin plugin: add role/ban fields to user, impersonatedBy to session.

ALTER TABLE "user" ADD COLUMN IF NOT EXISTS "role" text DEFAULT 'user';
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS "banned" boolean DEFAULT false;
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS "banReason" text;
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS "banExpires" timestamptz;

ALTER TABLE "session" ADD COLUMN IF NOT EXISTS "impersonatedBy" text;
