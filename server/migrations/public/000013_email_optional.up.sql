-- Identity overhaul: username/phone are the primary login identities; email is now
-- OPTIONAL. Historically platform_user.email was NOT NULL UNIQUE (see 000002), which
-- forced every user to carry a (often synthetic) email. We:
--   1. Drop the NOT NULL on email so users can be created with username+phone only.
--   2. Replace the plain UNIQUE constraint/index with a PARTIAL unique index that only
--      enforces uniqueness when email IS NOT NULL — so multiple email-less users coexist
--      while real emails stay unique.
-- Idempotent: safe whether or not the column is already nullable / the old constraint
-- still exists (the live dev DB may be in either state).

ALTER TABLE public.platform_user ALTER COLUMN email DROP NOT NULL;

-- Drop the legacy full UNIQUE constraint (000002 created it as platform_user_email_key,
-- which Postgres backs with an index of the same name). Drop both forms defensively.
ALTER TABLE public.platform_user DROP CONSTRAINT IF EXISTS platform_user_email_key;
DROP INDEX IF EXISTS public.platform_user_email_key;

-- Partial unique index: uniqueness only among rows that actually have an email.
CREATE UNIQUE INDEX IF NOT EXISTS platform_user_email_uniq
    ON public.platform_user (email)
    WHERE email IS NOT NULL;
