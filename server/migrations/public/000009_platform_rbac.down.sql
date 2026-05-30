DELETE FROM public.platform_setting WHERE key = 'governance_mode';
DELETE FROM public.role
 WHERE tenant_id IS NULL AND code IN ('super_admin','sys_admin','sec_admin','audit_admin');
DROP TABLE IF EXISTS public.platform_audit_log;
DROP TABLE IF EXISTS public.platform_setting;
DROP TABLE IF EXISTS public.platform_role_grant;
DROP TABLE IF EXISTS public.platform_permission;
