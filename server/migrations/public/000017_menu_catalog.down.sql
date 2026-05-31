DELETE FROM public.role_policy
WHERE resource = 'menu'
  AND action = 'write'
  AND role_id IN (
      SELECT id FROM public.role
      WHERE tenant_id IS NULL AND code = 'sys_admin'
  );

DROP TABLE IF EXISTS public.tenant_menu;
DROP TABLE IF EXISTS public.menu_catalog;
