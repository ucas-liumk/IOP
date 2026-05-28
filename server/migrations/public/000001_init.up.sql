-- v3.1 base tables. M2 will add tenant / platform_user / membership.

CREATE TABLE IF NOT EXISTS public.migration_history (
    id            UUID PRIMARY KEY,
    scope         TEXT NOT NULL,          -- 'public' or tenant slug
    migration_id  TEXT NOT NULL,          -- '000001_init'
    applied_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    checksum      TEXT NOT NULL,
    UNIQUE (scope, migration_id)
);

CREATE INDEX IF NOT EXISTS migration_history_applied_at_idx
    ON public.migration_history (applied_at DESC);

-- Marker row so cmd/migrate can verify a fresh DB.
INSERT INTO public.migration_history (id, scope, migration_id, checksum)
VALUES (
    gen_random_uuid(),
    'public',
    '000001_init',
    'placeholder-m1'
)
ON CONFLICT DO NOTHING;
