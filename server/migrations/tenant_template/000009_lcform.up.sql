-- Low-code online forms (在线表单 / 简道云·金数据·Airtable-style) tables — per tenant.
-- Aggregates: lcform_def (a form definition with a JSONB field schema) and
-- lcform_entry (one submitted record, data stored as JSONB keyed by field key).

CREATE TABLE IF NOT EXISTS lcform_def (
    id          UUID PRIMARY KEY,
    code        TEXT NOT NULL,                 -- short slug, unique per tenant
    name        TEXT NOT NULL,
    icon        TEXT NOT NULL DEFAULT '',       -- emoji / svg path / short label
    fields      JSONB NOT NULL DEFAULT '[]',    -- [{key,label,type,required,options}]
    status      TEXT NOT NULL DEFAULT 'active', -- active / archived
    created_by  UUID NOT NULL,                  -- member id
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (code)
);
CREATE INDEX IF NOT EXISTS lcform_def_status_idx ON lcform_def(status, created_at DESC);

CREATE TABLE IF NOT EXISTS lcform_entry (
    id            UUID PRIMARY KEY,
    form_id       UUID NOT NULL REFERENCES lcform_def(id) ON DELETE CASCADE,
    data          JSONB NOT NULL DEFAULT '{}',  -- {fieldKey: value}
    submitted_by  UUID NOT NULL,                -- member id
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS lcform_entry_form_idx ON lcform_entry(form_id, created_at DESC);
