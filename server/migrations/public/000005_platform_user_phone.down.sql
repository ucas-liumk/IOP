-- Reverting: enforce email NOT NULL again and drop phone.
-- Users can now exist without an email (phone-only / seeded admin), so backfill a
-- unique placeholder for any NULL email BEFORE re-adding the NOT NULL constraint,
-- otherwise the ALTER would fail. The id-based suffix keeps the email UNIQUE index happy.
UPDATE public.platform_user
   SET email = id::text || '@placeholder.invalid'
 WHERE email IS NULL;

ALTER TABLE public.platform_user ALTER COLUMN email SET NOT NULL;
DROP INDEX IF EXISTS public.platform_user_phone_key;
ALTER TABLE public.platform_user DROP COLUMN IF EXISTS phone;
