-- Route-level platform RBAC now uses coarse read/write actions for common
-- resources. Backfill equivalent write/manage aliases for existing built-in
-- platform roles so their historical create/update/disable policies continue to
-- authorize the same screens.

INSERT INTO public.role_policy (role_id, resource, action, effect)
SELECT r.id, v.resource, v.action, 'allow'
FROM public.role r
CROSS JOIN (VALUES
  ('org','write'),
  ('user','write')
) AS v(resource, action)
WHERE r.tenant_id IS NULL AND r.code = 'sys_admin'
ON CONFLICT (role_id, resource, action) DO NOTHING;

INSERT INTO public.role_policy (role_id, resource, action, effect)
SELECT r.id, v.resource, v.action, 'allow'
FROM public.role r
CROSS JOIN (VALUES
  ('role','read'),
  ('role','write')
) AS v(resource, action)
WHERE r.tenant_id IS NULL AND r.code = 'sec_admin'
ON CONFLICT (role_id, resource, action) DO NOTHING;
