-- Tenant-level display metadata for AppCenter. This is intentionally separate
-- from tenant_app so admins can classify an app without enabling/disabling it.
CREATE TABLE IF NOT EXISTS public.tenant_app_display_config (
    tenant_id  UUID NOT NULL REFERENCES public.tenant(id) ON DELETE CASCADE,
    app_code   TEXT NOT NULL,
    category   TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by UUID,
    PRIMARY KEY (tenant_id, app_code)
);

CREATE INDEX IF NOT EXISTS tenant_app_display_config_tenant_idx
    ON public.tenant_app_display_config(tenant_id);
