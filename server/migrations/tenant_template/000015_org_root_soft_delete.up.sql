-- Tenant schemas still carry tenant_id on ordinary organization rows so exports,
-- future cross-schema jobs, and integrity checks can distinguish tenant-owned data
-- without trusting a caller-provided tenant id.

ALTER TABLE department ADD COLUMN IF NOT EXISTS tenant_id UUID;
ALTER TABLE department ADD COLUMN IF NOT EXISTS is_root BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE department ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

UPDATE department
SET tenant_id = t.id
FROM public.tenant t
WHERE department.tenant_id IS NULL
  AND t.schema_name = current_schema();

INSERT INTO department (id, tenant_id, name, parent_id, order_num, status, is_root)
SELECT gen_random_uuid(), t.id, COALESCE(NULLIF(t.name, ''), t.slug), NULL, 0, 'active', TRUE
FROM public.tenant t
WHERE t.schema_name = current_schema()
  AND NOT EXISTS (
    SELECT 1 FROM department d
    WHERE d.tenant_id = t.id AND d.is_root = TRUE AND d.deleted_at IS NULL
  );

WITH root AS (
  SELECT d.id, d.tenant_id
  FROM department d
  WHERE d.is_root = TRUE AND d.deleted_at IS NULL
  ORDER BY d.created_at ASC
  LIMIT 1
)
UPDATE department d
SET parent_id = root.id
FROM root
WHERE d.tenant_id = root.tenant_id
  AND d.id <> root.id
  AND d.parent_id IS NULL
  AND d.deleted_at IS NULL;

ALTER TABLE department ALTER COLUMN tenant_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS department_tenant_deleted_idx ON department(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS department_deleted_idx ON department(deleted_at);
CREATE UNIQUE INDEX IF NOT EXISTS department_one_root_per_tenant_idx
  ON department(tenant_id)
  WHERE is_root = TRUE AND deleted_at IS NULL;

ALTER TABLE post ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS post_deleted_idx ON post(deleted_at);

ALTER TABLE notice ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS notice_deleted_idx ON notice(deleted_at);
