-- Per-user app workspace. Each row = "platform user X has added app Y to their
-- workspace in tenant Z", with an explicit ordering. This is a layer ON TOP of
-- public.tenant_app: tenant_app decides org-level availability (admin-gated);
-- user_app lets any logged-in member curate + order their own left rail.
-- App codes match Module.Manifest().Code (no FK because modules are code, not data).
CREATE TABLE IF NOT EXISTS public.user_app (
    platform_user_id UUID NOT NULL,
    tenant_id        UUID NOT NULL,
    app_code         TEXT NOT NULL,
    order_num        INT NOT NULL DEFAULT 0,
    added_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (platform_user_id, tenant_id, app_code)
);
CREATE INDEX IF NOT EXISTS user_app_user_order_idx
    ON public.user_app (platform_user_id, tenant_id, order_num);
