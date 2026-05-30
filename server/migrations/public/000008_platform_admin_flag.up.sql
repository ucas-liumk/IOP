-- Make platform_admin a GLOBAL identity attribute of a platform_user, decoupled
-- from any tenant membership/role. Previously "platform admin" was a per-tenant
-- role grant (held inside the bootstrap "system" tenant), which meant a platform
-- admin lost their powers outside that tenant. The flag below is the source of
-- truth going forward; the legacy platform_admin role grants are left in place
-- but no longer consulted by auth.
ALTER TABLE public.platform_user
    ADD COLUMN IF NOT EXISTS is_platform_admin BOOLEAN NOT NULL DEFAULT FALSE;

-- Backfill: anyone currently holding the platform_admin role (via any membership)
-- becomes a global platform admin.
UPDATE public.platform_user u
   SET is_platform_admin = TRUE
 WHERE u.id IN (
   SELECT m.platform_user_id
   FROM public.role_grant g
   JOIN public.role r ON r.id = g.role_id
   JOIN public.tenant_membership m ON m.member_id = g.member_id
   WHERE r.code = 'platform_admin'
 );

-- Safety net: the seeded bootstrap admin is always a platform admin.
UPDATE public.platform_user SET is_platform_admin = TRUE WHERE username = 'admin';
