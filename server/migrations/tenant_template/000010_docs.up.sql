-- Knowledge base (知识库 / 语雀·飞书文档·Notion-style) — per tenant schema.
-- A wiki is a tree of nodes: a node is either a 'folder' (container) or a 'doc'
-- (a page carrying html/markdown content). Lives INSIDE each tenant_<slug>
-- schema, so no schema prefix and no tenant_id column — schema isolation handles
-- that. Applied to every existing tenant schema on boot via SyncAllSchemas.

CREATE TABLE IF NOT EXISTS doc_node (
    id          UUID PRIMARY KEY,
    parent_id   UUID REFERENCES doc_node(id) ON DELETE CASCADE, -- NULL = root node
    title       TEXT NOT NULL,
    type        TEXT NOT NULL DEFAULT 'doc',      -- 'doc' | 'folder'
    content     TEXT NOT NULL DEFAULT '',         -- html/markdown body (docs only)
    order_num   INT  NOT NULL DEFAULT 0,
    created_by  UUID,
    updated_by  UUID,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS doc_node_parent_idx ON doc_node(parent_id, order_num);
CREATE INDEX IF NOT EXISTS doc_node_type_idx   ON doc_node(type);
