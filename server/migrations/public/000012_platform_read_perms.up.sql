-- Align the built-in platform template roles (sys/sec/audit_admin) with the P3
-- platform-console menu perms + route gates. super_admin already holds '*'/'*'
-- (000010) so it is unaffected. These rows only add read-side / notice perms that
-- the new menus + GET routes require; manage perms were already seeded in 000010.

-- sys_admin: system config surfaces (menu/dict/param/notice + job) read & notice manage.
INSERT INTO public.role_policy (role_id, resource, action, effect)
SELECT r.id, v.resource, v.action, 'allow'
FROM public.role r
CROSS JOIN (VALUES
  ('menu','read'),
  ('dict','read'),
  ('param','read'),
  ('notice','read'),('notice','manage'),
  ('job','read')
) AS v(resource, action)
WHERE r.tenant_id IS NULL AND r.code = 'sys_admin'
ON CONFLICT (role_id, resource, action) DO NOTHING;

-- audit_admin: logs + read-only monitor/session already; add menu read so the
-- monitoring section is reachable. (audit:read already seeded in 000010.)
INSERT INTO public.role_policy (role_id, resource, action, effect)
SELECT r.id, v.resource, v.action, 'allow'
FROM public.role r
CROSS JOIN (VALUES
  ('menu','read')
) AS v(resource, action)
WHERE r.tenant_id IS NULL AND r.code = 'audit_admin'
ON CONFLICT (role_id, resource, action) DO NOTHING;

-- sec_admin: menu read for navigation (role:manage + session already seeded).
INSERT INTO public.role_policy (role_id, resource, action, effect)
SELECT r.id, v.resource, v.action, 'allow'
FROM public.role r
CROSS JOIN (VALUES
  ('menu','read')
) AS v(resource, action)
WHERE r.tenant_id IS NULL AND r.code = 'sec_admin'
ON CONFLICT (role_id, resource, action) DO NOTHING;
