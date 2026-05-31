-- Mind-map (思维导图 / ProcessOn-style) tables — per tenant.
-- A mindmap is a titled document whose body is a JSONB node tree as produced by
-- the frontend editor (simple-mind-map): {data:{text},children:[...]}.

CREATE TABLE IF NOT EXISTS mindmap (
    id          UUID PRIMARY KEY,
    created_by  UUID NOT NULL,                              -- member id (owner)
    title       TEXT NOT NULL,
    data        JSONB NOT NULL DEFAULT '{"data":{"text":""},"children":[]}'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS mindmap_owner_idx ON mindmap(created_by, updated_at DESC);
