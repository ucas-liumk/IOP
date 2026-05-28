-- v3.1 M2: Tenancy + IAM tables in public schema.

CREATE TABLE IF NOT EXISTS public.tenant (
    id             UUID PRIMARY KEY,
    slug           TEXT NOT NULL UNIQUE,
    name           TEXT NOT NULL,
    schema_name    TEXT NOT NULL UNIQUE,
    status         TEXT NOT NULL DEFAULT 'active',  -- active / suspended / closed
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    suspended_at   TIMESTAMPTZ,
    closed_at      TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS tenant_status_idx ON public.tenant(status);

CREATE TABLE IF NOT EXISTS public.platform_user (
    id              UUID PRIMARY KEY,
    email           TEXT NOT NULL UNIQUE,
    password_hash   TEXT NOT NULL,
    mfa_secret      TEXT,
    status          TEXT NOT NULL DEFAULT 'active',
    last_login_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.tenant_membership (
    platform_user_id UUID NOT NULL REFERENCES public.platform_user(id),
    tenant_id        UUID NOT NULL REFERENCES public.tenant(id),
    member_id        UUID NOT NULL,
    joined_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    status           TEXT NOT NULL DEFAULT 'active',
    PRIMARY KEY (platform_user_id, tenant_id)
);
CREATE INDEX IF NOT EXISTS tenant_membership_tenant_idx ON public.tenant_membership(tenant_id);

CREATE TABLE IF NOT EXISTS public.session (
    id                UUID PRIMARY KEY,
    platform_user_id  UUID NOT NULL REFERENCES public.platform_user(id),
    tenant_id         UUID,
    member_id         UUID,
    issued_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at        TIMESTAMPTZ NOT NULL,
    revoked           BOOLEAN NOT NULL DEFAULT FALSE,
    ip_address        INET,
    user_agent        TEXT
);
CREATE INDEX IF NOT EXISTS session_user_idx ON public.session(platform_user_id);

CREATE TABLE IF NOT EXISTS public.role (
    id           UUID PRIMARY KEY,
    tenant_id    UUID,                                -- NULL = 平台级角色
    code         TEXT NOT NULL,
    name         TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, code)
);

CREATE TABLE IF NOT EXISTS public.role_policy (
    role_id      UUID NOT NULL REFERENCES public.role(id) ON DELETE CASCADE,
    resource     TEXT NOT NULL,
    action       TEXT NOT NULL,
    effect       TEXT NOT NULL DEFAULT 'allow',
    PRIMARY KEY (role_id, resource, action)
);

CREATE TABLE IF NOT EXISTS public.role_grant (
    role_id      UUID NOT NULL REFERENCES public.role(id) ON DELETE CASCADE,
    member_id    UUID NOT NULL,
    tenant_id    UUID NOT NULL,
    granted_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    granted_by   UUID,
    PRIMARY KEY (role_id, member_id)
);
CREATE INDEX IF NOT EXISTS role_grant_member_idx ON public.role_grant(tenant_id, member_id);

-- Built-in roles (created without tenant_id = platform-wide).
INSERT INTO public.role (id, tenant_id, code, name)
VALUES
  (gen_random_uuid(), NULL, 'platform_admin', '平台管理员'),
  (gen_random_uuid(), NULL, 'tenant_admin',   '租户管理员'),
  (gen_random_uuid(), NULL, 'tenant_member',  '租户成员')
ON CONFLICT DO NOTHING;
