DROP TABLE IF EXISTS public.role_dept;
ALTER TABLE public.role DROP COLUMN IF EXISTS data_scope;
ALTER TABLE public.role DROP COLUMN IF EXISTS is_builtin;
