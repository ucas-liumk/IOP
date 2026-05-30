DELETE FROM public.role_policy
 WHERE effect = 'allow'
   AND role_id IN (SELECT id FROM public.role WHERE tenant_id IS NULL AND code IN ('sys_admin','sec_admin','audit_admin'))
   AND (
     (resource = 'menu'   AND action = 'read')
     OR (resource = 'dict'   AND action = 'read')
     OR (resource = 'param'  AND action = 'read')
     OR (resource = 'notice' AND action IN ('read','manage'))
     OR (resource = 'job'    AND action = 'read')
   );
