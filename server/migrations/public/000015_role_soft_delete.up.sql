ALTER TABLE public.role ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS role_deleted_idx ON public.role(deleted_at);
