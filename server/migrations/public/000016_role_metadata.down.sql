DROP INDEX IF EXISTS public.role_tenant_code_uidx;
DROP INDEX IF EXISTS public.role_platform_code_uidx;
DROP INDEX IF EXISTS public.role_status_idx;

ALTER TABLE public.role DROP COLUMN IF EXISTS remark;
ALTER TABLE public.role DROP COLUMN IF EXISTS order_num;
ALTER TABLE public.role DROP COLUMN IF EXISTS status;
