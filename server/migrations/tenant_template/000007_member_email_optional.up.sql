-- Identity overhaul (per-tenant): the tenant `member` projection historically had
-- email NOT NULL (see 000001_member). Email is now OPTIONAL — a member created from a
-- username+phone-only platform_user has no email. Drop the NOT NULL so JoinMember can
-- insert NULL email. member.email was only ever a non-unique index, so there is no
-- unique constraint to relax. Idempotent (safe to re-run on every SyncAllSchemas).

ALTER TABLE member ALTER COLUMN email DROP NOT NULL;

-- Keep a lookup index for the optional secondary field.
CREATE INDEX IF NOT EXISTS member_email_idx ON member(email);
