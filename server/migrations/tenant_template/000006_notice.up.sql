-- Notices / announcements (通知公告) per tenant schema. Lives INSIDE each
-- tenant_<slug> schema, so no schema prefix and no tenant_id column — schema
-- isolation handles that. Applied to every existing tenant schema on boot via
-- SyncAllSchemas.

CREATE TABLE IF NOT EXISTS notice (
    id         UUID PRIMARY KEY,
    title      TEXT NOT NULL,
    content    TEXT NOT NULL DEFAULT '',
    type       TEXT NOT NULL DEFAULT 'notice',   -- 'notice' | 'announcement'
    status     TEXT NOT NULL DEFAULT 'draft',    -- 'draft' | 'published'
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS notice_status_idx  ON notice(status);
CREATE INDEX IF NOT EXISTS notice_created_idx ON notice(created_at DESC);
