-- Tenant-scoped app enablement. Each row = "tenant X has app Y installed".
-- App codes match Module.Manifest().Code (no FK because modules are code, not data).
CREATE TABLE IF NOT EXISTS public.tenant_app (
    tenant_id     UUID NOT NULL REFERENCES public.tenant(id) ON DELETE CASCADE,
    app_code      TEXT NOT NULL,
    installed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    installed_by  UUID,                                       -- platform_user_id of who installed
    config        JSONB NOT NULL DEFAULT '{}'::jsonb,         -- per-tenant app config (M5+)
    PRIMARY KEY (tenant_id, app_code)
);
CREATE INDEX IF NOT EXISTS tenant_app_tenant_idx ON public.tenant_app(tenant_id);
