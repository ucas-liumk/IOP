CREATE TABLE IF NOT EXISTS public.menu_catalog (
    id               UUID PRIMARY KEY,
    menu_key         TEXT NOT NULL UNIQUE,
    title            TEXT NOT NULL,
    menu_type        TEXT NOT NULL DEFAULT 'menu',
    parent_key       TEXT,
    route_path       TEXT,
    component_path   TEXT,
    permission_code  TEXT,
    icon             TEXT,
    order_num        INTEGER NOT NULL DEFAULT 0,
    visible          BOOLEAN NOT NULL DEFAULT TRUE,
    cacheable        BOOLEAN NOT NULL DEFAULT FALSE,
    status           TEXT NOT NULL DEFAULT 'active',
    app_code         TEXT,
    console          TEXT NOT NULL DEFAULT 'tenant',
    external_url     TEXT,
    iframe_url       TEXT,
    micro_app_code   TEXT,
    micro_entry      TEXT,
    is_builtin       BOOLEAN NOT NULL DEFAULT FALSE,
    source           TEXT NOT NULL DEFAULT 'manual',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at       TIMESTAMPTZ,
    CONSTRAINT menu_catalog_type_chk CHECK (menu_type IN ('dir','menu','button','link','iframe','micro')),
    CONSTRAINT menu_catalog_console_chk CHECK (console IN ('platform','tenant','both')),
    CONSTRAINT menu_catalog_status_chk CHECK (status IN ('active','disabled')),
    CONSTRAINT menu_catalog_button_perm_chk CHECK (menu_type <> 'button' OR COALESCE(permission_code,'') <> '')
);

CREATE INDEX IF NOT EXISTS menu_catalog_parent_idx ON public.menu_catalog(parent_key) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS menu_catalog_console_idx ON public.menu_catalog(console) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS menu_catalog_status_idx ON public.menu_catalog(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS menu_catalog_app_idx ON public.menu_catalog(app_code) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS public.tenant_menu (
    tenant_id   UUID NOT NULL REFERENCES public.tenant(id),
    menu_id     UUID NOT NULL REFERENCES public.menu_catalog(id) ON DELETE CASCADE,
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    order_num   INTEGER,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, menu_id)
);

CREATE INDEX IF NOT EXISTS tenant_menu_tenant_idx ON public.tenant_menu(tenant_id);
CREATE INDEX IF NOT EXISTS tenant_menu_menu_idx ON public.tenant_menu(menu_id);

INSERT INTO public.role_policy (role_id, resource, action, effect)
SELECT r.id, 'menu', 'write', 'allow'
FROM public.role r
WHERE r.tenant_id IS NULL AND r.code = 'sys_admin'
ON CONFLICT (role_id, resource, action) DO NOTHING;
