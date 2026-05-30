-- crm: 客户管理 CRM
-- Seed table. Add columns / additional tables as the domain grows.
CREATE TABLE IF NOT EXISTS crm_item (
    id          UUID PRIMARY KEY,
    title       TEXT NOT NULL,
    body        TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
