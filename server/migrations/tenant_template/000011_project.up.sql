-- Project management (项目管理 / Teambition / Trello / Jira-style Kanban) tables — per tenant.
-- Aggregates: Project (a board), Column (a lane), Card (a ticket on a lane).

CREATE TABLE IF NOT EXISTS project (
    id           UUID PRIMARY KEY,
    name         TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'active',  -- active / archived
    created_by   UUID NOT NULL,                   -- member id of the creator
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS project_status_idx ON project(status, created_at);

CREATE TABLE IF NOT EXISTS project_column (
    id           UUID PRIMARY KEY,
    project_id   UUID NOT NULL REFERENCES project(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    order_num    INT  NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS project_column_project_idx ON project_column(project_id, order_num);

CREATE TABLE IF NOT EXISTS project_card (
    id           UUID PRIMARY KEY,
    project_id   UUID NOT NULL REFERENCES project(id) ON DELETE CASCADE,
    column_id    UUID NOT NULL REFERENCES project_column(id) ON DELETE CASCADE,
    title        TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    assignee_id  UUID,                            -- member id (nullable = unassigned)
    due_date     TIMESTAMPTZ,
    priority     SMALLINT NOT NULL DEFAULT 0,     -- 0 none, 1 low, 2 medium, 3 high
    order_num    INT  NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS project_card_project_idx ON project_card(project_id);
CREATE INDEX IF NOT EXISTS project_card_column_idx ON project_card(column_id, order_num);
CREATE INDEX IF NOT EXISTS project_card_assignee_idx ON project_card(assignee_id);
