DROP INDEX IF EXISTS notice_deleted_idx;
ALTER TABLE notice DROP COLUMN IF EXISTS deleted_at;

DROP INDEX IF EXISTS post_deleted_idx;
ALTER TABLE post DROP COLUMN IF EXISTS deleted_at;

DROP INDEX IF EXISTS department_one_root_per_tenant_idx;
DROP INDEX IF EXISTS department_deleted_idx;
DROP INDEX IF EXISTS department_tenant_deleted_idx;
ALTER TABLE department DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE department DROP COLUMN IF EXISTS is_root;
ALTER TABLE department DROP COLUMN IF EXISTS tenant_id;
