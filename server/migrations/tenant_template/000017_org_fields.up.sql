ALTER TABLE department ADD COLUMN IF NOT EXISTS org_code TEXT;
ALTER TABLE department ADD COLUMN IF NOT EXISTS org_type TEXT NOT NULL DEFAULT 'department';
ALTER TABLE department ADD COLUMN IF NOT EXISTS leader_account TEXT;
ALTER TABLE department ADD COLUMN IF NOT EXISTS remark TEXT;

UPDATE department
SET org_code = CASE
    WHEN COALESCE(is_root, FALSE) THEN 'ROOT'
    ELSE 'ORG-' || replace(id::text, '-', '')::text
  END
WHERE org_code IS NULL OR btrim(org_code) = '';

UPDATE department
SET org_type = 'unit'
WHERE COALESCE(is_root, FALSE);

ALTER TABLE department ALTER COLUMN org_code SET NOT NULL;
ALTER TABLE department ALTER COLUMN org_type SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS department_tenant_org_code_uidx
  ON department(tenant_id, lower(org_code))
  WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS department_tenant_status_idx
  ON department(tenant_id, status)
  WHERE deleted_at IS NULL;
