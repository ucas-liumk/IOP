DELETE FROM public.role_policy
WHERE (resource, action) IN (('org','write'), ('user','write'))
  AND role_id IN (SELECT id FROM public.role WHERE tenant_id IS NULL AND code = 'sys_admin');

DELETE FROM public.role_policy
WHERE (resource, action) IN (('role','read'), ('role','write'))
  AND role_id IN (SELECT id FROM public.role WHERE tenant_id IS NULL AND code = 'sec_admin');
