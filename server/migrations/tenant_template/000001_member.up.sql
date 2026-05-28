-- Per-tenant schema baseline. Applied to every tenant_<slug> by SchemaProvisioner.

CREATE TABLE IF NOT EXISTS member (
    id             UUID PRIMARY KEY,
    platform_user_id UUID NOT NULL,
    display_name   TEXT NOT NULL,
    email          TEXT NOT NULL,
    avatar_url     TEXT,
    department     TEXT,
    title          TEXT,
    phone          TEXT,
    profile        JSONB NOT NULL DEFAULT '{}'::jsonb,
    joined_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    status         TEXT NOT NULL DEFAULT 'active',
    UNIQUE (platform_user_id)
);
CREATE INDEX IF NOT EXISTS member_email_idx ON member(email);
CREATE INDEX IF NOT EXISTS member_department_idx ON member(department);
