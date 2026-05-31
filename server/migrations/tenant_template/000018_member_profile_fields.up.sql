ALTER TABLE member ADD COLUMN IF NOT EXISTS gender TEXT;
ALTER TABLE member ADD COLUMN IF NOT EXISTS remark TEXT;

CREATE INDEX IF NOT EXISTS member_status_idx ON member(status);
CREATE INDEX IF NOT EXISTS member_dept_id_idx ON member(dept_id);
