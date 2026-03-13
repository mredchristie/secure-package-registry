-- Manually Written Down Script

-- Going in reverse order of the up script: Drop Indexes first
DROP INDEX IF EXISTS invitation_email_idx;
DROP INDEX IF EXISTS invitation_organizationId_idx;
DROP INDEX IF EXISTS member_userId_idx;
DROP INDEX IF EXISTS member_organizationId_idx;
DROP INDEX IF EXISTS organization_slug_uidx;
DROP INDEX IF EXISTS verification_identifier_idx;
DROP INDEX IF EXISTS account_userId_idx;
DROP INDEX IF EXISTS session_userId_idx;

-- Then drop tables

DROP TABLE IF EXISTS invitation;
DROP TABLE IF EXISTS member;
DROP TABLE IF EXISTS organization;
DROP TABLE IF EXISTS verification;
DROP TABLE IF EXISTS account;
DROP TABLE IF EXISTS session;

-- Remove Added Columns

ALTER TABLE "user" DROP COLUMN IF EXISTS "updatedAt";
ALTER TABLE "user" DROP COLUMN IF EXISTS "createdAt";
ALTER TABLE "user" DROP COLUMN IF EXISTS "image";
ALTER TABLE "user" DROP COLUMN IF EXISTS "emailVerified";
ALTER TABLE "user" DROP COLUMN IF EXISTS "name";
