-- Task management (任务清单 / TickTick-style) tables — per tenant.
-- Aggregates: TaskList (a list/project) and Task (a to-do, optionally a subtask).

CREATE TABLE IF NOT EXISTS task_list (
    id          UUID PRIMARY KEY,
    owner       UUID NOT NULL,                 -- member id
    name        TEXT NOT NULL,
    color       TEXT NOT NULL DEFAULT '',       -- CSS color for the dot
    sort_order  INT  NOT NULL DEFAULT 0,
    archived    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS task_list_owner_idx ON task_list(owner, archived, sort_order);

CREATE TABLE IF NOT EXISTS task (
    id            UUID PRIMARY KEY,
    owner         UUID NOT NULL,               -- member id (the assignee/creator)
    list_id       UUID REFERENCES task_list(id) ON DELETE CASCADE,
    parent_id     UUID REFERENCES task(id) ON DELETE CASCADE, -- subtask parent
    title         TEXT NOT NULL,
    note          TEXT NOT NULL DEFAULT '',
    priority      SMALLINT NOT NULL DEFAULT 0, -- 0 none, 1 low, 2 medium, 3 high
    status        TEXT NOT NULL DEFAULT 'todo', -- todo / done
    due_date      TIMESTAMPTZ,
    completed_at  TIMESTAMPTZ,
    tags          TEXT[] NOT NULL DEFAULT '{}',
    sort_order    INT  NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS task_owner_status_idx ON task(owner, status);
CREATE INDEX IF NOT EXISTS task_list_idx ON task(list_id);
CREATE INDEX IF NOT EXISTS task_owner_due_idx ON task(owner, due_date);
CREATE INDEX IF NOT EXISTS task_parent_idx ON task(parent_id);
