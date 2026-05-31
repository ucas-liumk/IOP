-- Generic RBAC core: drop the 三员分立 / governance_mode special-casing in favour of
-- plain user→role→(menu permission + data scope). Built-in admin roles get all-access
-- via a wildcard role_policy (config, not code), so removing the Enforce short-circuit
-- keeps them omnipotent.

-- Data scope + built-in marker on roles (data scope is reserved; not yet enforced on
-- business queries — see spec §0.4 / §9).
ALTER TABLE public.role ADD COLUMN IF NOT EXISTS data_scope TEXT NOT NULL DEFAULT 'all';
ALTER TABLE public.role ADD COLUMN IF NOT EXISTS is_builtin BOOLEAN NOT NULL DEFAULT FALSE;

-- Custom data-scope dept binding (reserved; application-layer check later).
CREATE TABLE IF NOT EXISTS public.role_dept (
    role_id   UUID NOT NULL REFERENCES public.role(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    dept_id   UUID NOT NULL,
    PRIMARY KEY (role_id, dept_id)
);

-- Mark the built-in roles (tenant_id IS NULL = platform-wide templates).
UPDATE public.role SET is_builtin = TRUE
 WHERE tenant_id IS NULL AND code IN ('super_admin','platform_admin','tenant_admin','tenant_member');

-- All-access wildcard for the admin roles. With the Enforce/EnforcePlatform code
-- short-circuit removed, this policy is what grants them full power.
INSERT INTO public.role_policy (role_id, resource, action, effect)
SELECT id, '*', '*', 'allow' FROM public.role
 WHERE tenant_id IS NULL AND code IN ('super_admin','platform_admin','tenant_admin')
ON CONFLICT (role_id, resource, action) DO NOTHING;

-- Keep the 三员 template roles meaningful: seed their default platform policies as
-- plain role_policy rows (same sets as the old code-level defaultRolePolicies). These
-- roles stay editable/deletable plain roles — no special handling anywhere.
-- sys_admin: org/user/membership + app/param/dict/announce/branding + monitoring.
INSERT INTO public.role_policy (role_id, resource, action, effect)
SELECT r.id, v.resource, v.action, 'allow'
FROM public.role r
CROSS JOIN (VALUES
  ('org','read'),('org','create'),('org','update'),('org','suspend'),('org','close'),('org','hierarchy'),
  ('user','read'),('user','create'),('user','update'),('user','disable'),('user','resetpwd'),('user','impersonate'),
  ('membership','assign'),
  ('app','manage'),('param','manage'),('dict','manage'),('announce','manage'),('branding','manage'),
  ('monitor','read'),('job','manage'),('cache','manage'),('schema','sync'),('codegen','use'),('backup','manage')
) AS v(resource, action)
WHERE r.tenant_id IS NULL AND r.code = 'sys_admin'
ON CONFLICT (role_id, resource, action) DO NOTHING;

-- sec_admin: authz/role/security + read-only org/user.
INSERT INTO public.role_policy (role_id, resource, action, effect)
SELECT r.id, v.resource, v.action, 'allow'
FROM public.role r
CROSS JOIN (VALUES
  ('org','read'),('user','read'),
  ('role','manage'),('authz','grant'),('platform_admin','grant'),
  ('security_policy','manage'),('session','read'),('session','revoke')
) AS v(resource, action)
WHERE r.tenant_id IS NULL AND r.code = 'sec_admin'
ON CONFLICT (role_id, resource, action) DO NOTHING;

-- audit_admin: audit/login_log + read-only org/user/session/monitor.
INSERT INTO public.role_policy (role_id, resource, action, effect)
SELECT r.id, v.resource, v.action, 'allow'
FROM public.role r
CROSS JOIN (VALUES
  ('org','read'),('user','read'),('session','read'),('monitor','read'),
  ('audit','read'),('audit','export'),('audit','config'),('audit','purge'),('login_log','read')
) AS v(resource, action)
WHERE r.tenant_id IS NULL AND r.code = 'audit_admin'
ON CONFLICT (role_id, resource, action) DO NOTHING;

-- Governance mode is gone; platform_setting stays as a generic KV param store.
DELETE FROM public.platform_setting WHERE key = 'governance_mode';
