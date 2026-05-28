-- Audit log per tenant
CREATE TABLE IF NOT EXISTS audit_log (
    id            UUID PRIMARY KEY,
    occurred_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    actor         TEXT NOT NULL,            -- member id or 'system'
    action        TEXT NOT NULL,            -- event topic
    resource      TEXT,
    resource_id   TEXT,
    trace_id      TEXT,
    detail        JSONB NOT NULL DEFAULT '{}'::jsonb
);
CREATE INDEX IF NOT EXISTS audit_log_occurred_idx ON audit_log(occurred_at DESC);
CREATE INDEX IF NOT EXISTS audit_log_actor_idx    ON audit_log(actor);
CREATE INDEX IF NOT EXISTS audit_log_action_idx   ON audit_log(action);

-- Notifications per tenant
CREATE TABLE IF NOT EXISTS notification (
    id           UUID PRIMARY KEY,
    recipient    UUID NOT NULL,            -- member id
    type         TEXT NOT NULL,            -- 'okr.weekly_overdue' etc.
    title        TEXT NOT NULL,
    body         TEXT,
    payload      JSONB NOT NULL DEFAULT '{}'::jsonb,
    read         BOOLEAN NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    read_at      TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS notif_recipient_idx ON notification(recipient, created_at DESC);
CREATE INDEX IF NOT EXISTS notif_unread_idx    ON notification(recipient) WHERE NOT read;

-- File attachments per tenant
CREATE TABLE IF NOT EXISTS attachment (
    id           UUID PRIMARY KEY,
    biz_module   TEXT NOT NULL,
    biz_id       TEXT NOT NULL,
    object_key   TEXT NOT NULL,
    name         TEXT NOT NULL,
    size         BIGINT NOT NULL,
    mime_type    TEXT,
    uploader     UUID NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS attachment_biz_idx ON attachment(biz_module, biz_id);

-- Dictionary tenant overrides
CREATE TABLE IF NOT EXISTS dict_override (
    type_code    TEXT NOT NULL,
    item_code    TEXT NOT NULL,
    override     JSONB NOT NULL DEFAULT '{}'::jsonb,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (type_code, item_code)
);
