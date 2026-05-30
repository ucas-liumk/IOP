-- Phone-first identity: allow signup with phone instead of email.
-- Email becomes optional; phone is added as a partial-unique field
-- (SMS verification will be wired in later).

ALTER TABLE public.platform_user
    ADD COLUMN IF NOT EXISTS phone TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS platform_user_phone_key
    ON public.platform_user (phone)
    WHERE phone IS NOT NULL;

ALTER TABLE public.platform_user
    ALTER COLUMN email DROP NOT NULL;
