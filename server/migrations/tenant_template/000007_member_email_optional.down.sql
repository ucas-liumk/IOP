-- Revert member.email back to NOT NULL. Fails if any NULL-email rows exist (by design).
ALTER TABLE member ALTER COLUMN email SET NOT NULL;
