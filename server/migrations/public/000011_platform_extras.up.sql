-- Platform-console extras (P3): notice / cron jobs. Params reuse the existing
-- public.platform_setting KV table (000009); operation/login logs reuse
-- public.platform_audit_log (000009); online users reuse public.session (000002);
-- monitor reuses the in-process health.Registry. No new tables for those.

-- Platform notice / announcement (平台通知公告). Distinct from tenant-scoped notices.
CREATE TABLE IF NOT EXISTS public.platform_notice (
    id         UUID PRIMARY KEY,
    title      TEXT NOT NULL,
    content    TEXT NOT NULL DEFAULT '',
    type       TEXT NOT NULL DEFAULT 'notice',   -- notice / bulletin / maintenance ...
    status     TEXT NOT NULL DEFAULT 'draft',     -- draft / published / withdrawn
    created_by UUID,                               -- no FK: preserve record if author removed
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS platform_notice_status_idx
    ON public.platform_notice (status, created_at DESC);

-- Scheduled jobs (定时任务, minimal). A lightweight in-process ticker is optional;
-- the data model + manual run-now is the contract. handler names a no-op/echo entry
-- in the in-code registry.
CREATE TABLE IF NOT EXISTS public.platform_job (
    id          UUID PRIMARY KEY,
    name        TEXT NOT NULL,
    cron_expr   TEXT NOT NULL DEFAULT '',
    handler     TEXT NOT NULL DEFAULT 'noop',
    status      TEXT NOT NULL DEFAULT 'enabled',   -- enabled / disabled
    last_run_at TIMESTAMPTZ,
    next_run_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.platform_job_run (
    id          UUID PRIMARY KEY,
    job_id      UUID NOT NULL REFERENCES public.platform_job(id) ON DELETE CASCADE,
    started_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ,
    status      TEXT NOT NULL DEFAULT 'running',    -- running / success / failed
    detail      TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS platform_job_run_job_idx
    ON public.platform_job_run (job_id, started_at DESC);
