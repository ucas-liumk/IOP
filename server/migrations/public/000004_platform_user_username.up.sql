-- Add a username column to support login-by-username.
-- Nullable so existing rows aren't broken; UNIQUE so duplicates are caught.

ALTER TABLE public.platform_user
    ADD COLUMN IF NOT EXISTS username TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS platform_user_username_key
    ON public.platform_user (username)
    WHERE username IS NOT NULL;

-- Demo account convenience: let "demo" log in alongside the email form.
UPDATE public.platform_user
   SET username = 'demo'
 WHERE email = 'demo@iop.test'
   AND username IS NULL;
