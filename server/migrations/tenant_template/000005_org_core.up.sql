-- Org-internal structure for each tenant schema (ruoyi-style): departments (tree) +
-- posts + member.dept_id + member↔post mapping. Lives INSIDE each tenant_<slug>
-- schema, so no schema prefix and no tenant_id column — schema isolation handles that.
-- Applied to every existing tenant schema on boot via SyncAllSchemas.

CREATE TABLE IF NOT EXISTS department (
    id         UUID PRIMARY KEY,
    name       TEXT NOT NULL,
    parent_id  UUID,
    order_num  INT NOT NULL DEFAULT 0,
    leader     TEXT,
    phone      TEXT,
    email      TEXT,
    status     TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS department_parent_idx ON department(parent_id);

CREATE TABLE IF NOT EXISTS post (
    id         UUID PRIMARY KEY,
    code       TEXT NOT NULL UNIQUE,
    name       TEXT NOT NULL,
    order_num  INT NOT NULL DEFAULT 0,
    status     TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE member ADD COLUMN IF NOT EXISTS dept_id UUID;

CREATE TABLE IF NOT EXISTS member_post (
    member_id UUID NOT NULL,
    post_id   UUID NOT NULL,
    PRIMARY KEY (member_id, post_id)
);
