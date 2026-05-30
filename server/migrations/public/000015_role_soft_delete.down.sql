DROP INDEX IF EXISTS public.role_deleted_idx;
ALTER TABLE public.role DROP COLUMN IF EXISTS deleted_at;
