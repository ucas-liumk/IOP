ALTER TABLE public.role ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active';
ALTER TABLE public.role ADD COLUMN IF NOT EXISTS order_num INTEGER NOT NULL DEFAULT 0;
ALTER TABLE public.role ADD COLUMN IF NOT EXISTS remark TEXT;

UPDATE public.role SET status = 'active' WHERE status IS NULL OR status = '';

CREATE INDEX IF NOT EXISTS role_status_idx
    ON public.role(status)
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS role_platform_code_uidx
    ON public.role(lower(code))
    WHERE tenant_id IS NULL AND deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS role_tenant_code_uidx
    ON public.role(tenant_id, lower(code))
    WHERE tenant_id IS NOT NULL AND deleted_at IS NULL;
