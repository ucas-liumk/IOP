DROP INDEX IF EXISTS department_tenant_status_idx;
DROP INDEX IF EXISTS department_tenant_org_code_uidx;

ALTER TABLE department DROP COLUMN IF EXISTS remark;
ALTER TABLE department DROP COLUMN IF EXISTS leader_account;
ALTER TABLE department DROP COLUMN IF EXISTS org_type;
ALTER TABLE department DROP COLUMN IF EXISTS org_code;
