-- Pending applications captured before a platform_user exists.
-- On approval: insert platform_user, JoinMember, grant role, mark approved.
-- On rejection: keep row for audit, no user created.

CREATE TABLE IF NOT EXISTS public.registration_application (
    id                UUID PRIMARY KEY,
    username          TEXT NOT NULL,
    real_name         TEXT NOT NULL,
    organization      TEXT NOT NULL,
    phone             TEXT,
    password_hash     TEXT NOT NULL,
    status            TEXT NOT NULL DEFAULT 'pending'
                          CHECK (status IN ('pending','approved','rejected')),
    applied_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    reviewed_by       UUID REFERENCES public.platform_user(id),
    reviewed_at       TIMESTAMPTZ,
    reject_reason     TEXT,
    target_tenant_id  UUID REFERENCES public.tenant(id),
    granted_role      TEXT
);

-- At most ONE pending application per username at a time.
CREATE UNIQUE INDEX IF NOT EXISTS registration_application_pending_username_key
    ON public.registration_application (username)
    WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS registration_application_status_idx
    ON public.registration_application (status, applied_at);
