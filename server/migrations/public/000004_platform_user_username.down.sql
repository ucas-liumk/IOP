DROP INDEX IF EXISTS public.platform_user_username_key;
ALTER TABLE public.platform_user DROP COLUMN IF EXISTS username;
