DROP INDEX IF EXISTS task_deleted_idx;
ALTER TABLE task DROP COLUMN IF EXISTS deleted_at;

DROP INDEX IF EXISTS task_list_deleted_idx;
ALTER TABLE task_list DROP COLUMN IF EXISTS deleted_at;
