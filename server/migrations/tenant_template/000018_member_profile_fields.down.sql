DROP INDEX IF EXISTS member_dept_id_idx;
DROP INDEX IF EXISTS member_status_idx;

ALTER TABLE member DROP COLUMN IF EXISTS remark;
ALTER TABLE member DROP COLUMN IF EXISTS gender;
