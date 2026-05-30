ALTER TABLE task_list ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS task_list_deleted_idx ON task_list(deleted_at);

ALTER TABLE task ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS task_deleted_idx ON task(deleted_at);
