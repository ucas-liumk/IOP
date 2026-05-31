-- Revert email back to NOT NULL UNIQUE. NOTE: this will FAIL if any email-less or
-- duplicate-email rows exist (by design — the down path assumes the data still
-- satisfies the old, stricter invariant).

DROP INDEX IF EXISTS public.platform_user_email_uniq;

ALTER TABLE public.platform_user ALTER COLUMN email SET NOT NULL;

ALTER TABLE public.platform_user
    ADD CONSTRAINT platform_user_email_key UNIQUE (email);
