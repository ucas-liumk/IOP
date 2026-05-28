-- OKR (工作安排) tables - per tenant.
-- Aggregate roots: Plan (contains PlanItem entities) + Report (contains ReportEntry).

CREATE TABLE IF NOT EXISTS okr_plan (
    id            UUID PRIMARY KEY,
    level         TEXT NOT NULL,            -- year / half_year / month / week
    owner         UUID NOT NULL,
    period_start  DATE NOT NULL,
    period_end    DATE NOT NULL,
    title         TEXT NOT NULL,
    parent_id     UUID,                     -- references parent plan (nullable for year)
    status        TEXT NOT NULL DEFAULT 'draft',  -- draft / active / closed
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (owner, level, period_start)
);
CREATE INDEX IF NOT EXISTS okr_plan_owner_idx ON okr_plan(owner, level, period_start);
CREATE INDEX IF NOT EXISTS okr_plan_parent_idx ON okr_plan(parent_id);

CREATE TABLE IF NOT EXISTS okr_plan_item (
    id            UUID PRIMARY KEY,
    plan_id       UUID NOT NULL REFERENCES okr_plan(id) ON DELETE CASCADE,
    title         TEXT NOT NULL,
    weight        INT  NOT NULL DEFAULT 0,
    progress_pct  INT  NOT NULL DEFAULT 0,
    progress_note TEXT,
    status        TEXT NOT NULL DEFAULT 'todo',  -- todo / doing / done / blocked
    sort_order    INT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS okr_plan_item_plan_idx ON okr_plan_item(plan_id);

CREATE TABLE IF NOT EXISTS okr_report (
    id            UUID PRIMARY KEY,
    type          TEXT NOT NULL,            -- daily / weekly
    owner         UUID NOT NULL,
    period_start  DATE NOT NULL,
    period_end    DATE NOT NULL,
    summary       TEXT,
    submitted_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (owner, type, period_start)
);
CREATE INDEX IF NOT EXISTS okr_report_owner_idx ON okr_report(owner, type, period_end DESC);

CREATE TABLE IF NOT EXISTS okr_report_entry (
    id            UUID PRIMARY KEY,
    report_id     UUID NOT NULL REFERENCES okr_report(id) ON DELETE CASCADE,
    plan_item_id  UUID,
    title         TEXT NOT NULL,
    detail        TEXT,
    progress_note TEXT,
    sort_order    INT NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS okr_report_entry_report_idx ON okr_report_entry(report_id);

CREATE TABLE IF NOT EXISTS okr_report_comment (
    id            UUID PRIMARY KEY,
    report_id     UUID NOT NULL REFERENCES okr_report(id) ON DELETE CASCADE,
    author        UUID NOT NULL,
    body          TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS okr_report_comment_report_idx ON okr_report_comment(report_id);
