-- Platform-level RBAC foundation (三员分立 / 等保).
-- Reuses public.role (tenant_id IS NULL = platform role) and public.role_policy.
-- Adds: a permission catalog, platform_user→role grants (the existing role_grant
-- is keyed by member×tenant and can't grant to a tenant-less platform user), a
-- settings table for the governance mode switch, and a platform-scoped audit log
-- (audit_log currently lives only inside tenant schemas).

-- Catalog of assignable platform permission points (resource/action). Upserted
-- from code at boot (see SeedPlatformRBAC). domain ∈ org_identity / security /
-- audit / app_config / monitoring.
CREATE TABLE IF NOT EXISTS public.platform_permission (
    resource     TEXT NOT NULL,
    action       TEXT NOT NULL,
    domain       TEXT NOT NULL,
    label        TEXT NOT NULL,
    is_high_risk BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (resource, action)
);

-- Grant a platform role (a role with tenant_id IS NULL) directly to a platform_user.
CREATE TABLE IF NOT EXISTS public.platform_role_grant (
    role_id          UUID NOT NULL REFERENCES public.role(id) ON DELETE CASCADE,
    platform_user_id UUID NOT NULL REFERENCES public.platform_user(id) ON DELETE CASCADE,
    granted_by       UUID,                          -- no FK: preserve audit trail if granter is deleted
    granted_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (role_id, platform_user_id)
);
CREATE INDEX IF NOT EXISTS platform_role_grant_user_idx
    ON public.platform_role_grant (platform_user_id);

-- Platform-wide settings (key→JSONB). First key: governance_mode.
CREATE TABLE IF NOT EXISTS public.platform_setting (
    key        TEXT PRIMARY KEY,
    value      JSONB NOT NULL,
    updated_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Platform-scoped audit (separate from tenant_<slug>.audit_log).
CREATE TABLE IF NOT EXISTS public.platform_audit_log (
    id              UUID PRIMARY KEY,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    actor           TEXT NOT NULL,
    actor_role      TEXT,
    action          TEXT NOT NULL,
    resource        TEXT,
    resource_id     TEXT,
    reason          TEXT,
    governance_mode TEXT,
    trace_id        TEXT,
    detail          JSONB
);
CREATE INDEX IF NOT EXISTS platform_audit_log_time_idx
    ON public.platform_audit_log (occurred_at DESC);

-- Built-in platform roles (tenant_id IS NULL = platform-wide). super_admin is the
-- bootstrap/super role; the three 等保 members are sys/sec/audit. Legacy
-- 'platform_admin' role (from 000002) is left untouched and unused.
INSERT INTO public.role (id, tenant_id, code, name)
SELECT gen_random_uuid(), NULL, v.code, v.name
FROM (VALUES
  ('super_admin',  '超级管理员'),
  ('sys_admin',    '系统管理员'),
  ('sec_admin',    '安全管理员'),
  ('audit_admin',  '审计管理员')
) AS v(code, name)
WHERE NOT EXISTS (
  SELECT 1 FROM public.role r WHERE r.tenant_id IS NULL AND r.code = v.code
);

-- Backfill: every existing global platform admin holds super_admin.
INSERT INTO public.platform_role_grant (role_id, platform_user_id, granted_by)
SELECT r.id, u.id, NULL
FROM public.platform_user u
CROSS JOIN public.role r
WHERE u.is_platform_admin = TRUE AND r.code = 'super_admin' AND r.tenant_id IS NULL
ON CONFLICT (role_id, platform_user_id) DO NOTHING;

-- Default governance mode: single_admin (super admin keeps full power; no break).
INSERT INTO public.platform_setting (key, value)
VALUES ('governance_mode', '"single_admin"'::jsonb)
ON CONFLICT (key) DO NOTHING;
