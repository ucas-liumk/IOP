-- Approval center (审批中心 / 钉钉审批·飞书审批-style) tables — per tenant.
-- Aggregates:
--   approval_form     — a reusable approval template (custom field schema + flow definition)
--   approval_instance — one submitted approval (snapshots the form's name/fields at submit time)
--   approval_task     — one approver/cc node-task generated for an instance node
-- Lives INSIDE each tenant_<slug> schema (schema isolation handles tenant scoping).

CREATE TABLE IF NOT EXISTS approval_form (
    id          UUID PRIMARY KEY,
    code        TEXT NOT NULL,                 -- short unique-ish code per tenant (e.g. "leave")
    name        TEXT NOT NULL,                 -- "请假申请"
    icon        TEXT NOT NULL DEFAULT '',      -- SVG path data (optional)
    description TEXT NOT NULL DEFAULT '',
    fields      JSONB NOT NULL DEFAULT '[]'::jsonb,  -- [{key,label,type,required,options[]}]
    flow        JSONB NOT NULL DEFAULT '[]'::jsonb,  -- [{type,assignee_type,assignee_id,role_code,mode}]
    status      TEXT NOT NULL DEFAULT 'active',      -- active / disabled
    created_by  UUID NOT NULL,                       -- member id
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS approval_form_status_idx ON approval_form(status, created_at DESC);

CREATE TABLE IF NOT EXISTS approval_instance (
    id           UUID PRIMARY KEY,
    form_id      UUID NOT NULL,                       -- soft ref to approval_form (snapshot kept here)
    form_name    TEXT NOT NULL,                       -- snapshot of the form name at submit time
    fields       JSONB NOT NULL DEFAULT '[]'::jsonb,  -- snapshot of the field schema
    data         JSONB NOT NULL DEFAULT '{}'::jsonb,  -- submitted values keyed by field key
    initiator_id UUID NOT NULL,                       -- member id
    status       TEXT NOT NULL DEFAULT 'pending',     -- pending / approved / rejected / canceled
    current_node INT  NOT NULL DEFAULT 0,             -- index into the snapshotted flow
    flow         JSONB NOT NULL DEFAULT '[]'::jsonb,  -- snapshot of the flow at submit time
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at  TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS approval_instance_initiator_idx ON approval_instance(initiator_id, created_at DESC);
CREATE INDEX IF NOT EXISTS approval_instance_status_idx ON approval_instance(status, created_at DESC);

CREATE TABLE IF NOT EXISTS approval_task (
    id           UUID PRIMARY KEY,
    instance_id  UUID NOT NULL REFERENCES approval_instance(id) ON DELETE CASCADE,
    node_index   INT  NOT NULL DEFAULT 0,             -- which flow node this task belongs to
    assignee_id  UUID NOT NULL,                       -- member id the task is assigned to
    type         TEXT NOT NULL DEFAULT 'approve',     -- approve / cc
    mode         TEXT NOT NULL DEFAULT 'or',          -- and / or (countersign vs any-one)
    status       TEXT NOT NULL DEFAULT 'pending',     -- pending / approved / rejected / read
    comment      TEXT NOT NULL DEFAULT '',
    acted_at     TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS approval_task_assignee_idx ON approval_task(assignee_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS approval_task_instance_idx ON approval_task(instance_id, node_index);

-- Self-heal: tenants provisioned from an earlier version of this migration may
-- predate the `flow` snapshot column. CREATE TABLE IF NOT EXISTS is a no-op on an
-- existing table and won't add it, so add it idempotently here. SyncAllSchemas
-- re-runs this file on every boot, bringing drifted tenants up to date.
ALTER TABLE approval_instance ADD COLUMN IF NOT EXISTS flow JSONB NOT NULL DEFAULT '[]'::jsonb;
