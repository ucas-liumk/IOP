-- Force a password change on first login (used for the seeded default admin and
-- any admin-created account whose initial password was set by someone else).
ALTER TABLE public.platform_user
    ADD COLUMN IF NOT EXISTS password_must_change BOOLEAN NOT NULL DEFAULT FALSE;
