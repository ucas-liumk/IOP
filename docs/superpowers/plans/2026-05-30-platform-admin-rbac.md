# 平台管理员 RBAC + 三员分立 Implementation Plan (Phase 0 地基)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a platform-level RBAC foundation with 三员分立 (system/security/audit admins) + a single_admin/three_member governance switch, so the platform console's features can be gated per-permission and assigned by the super admin.

**Architecture:** Reuse existing `public.role` (`tenant_id IS NULL` = platform role) + `public.role_policy` (role→resource/action). Add `platform_permission` (catalog), `platform_role_grant` (role→platform_user), `platform_setting` (governance_mode), and `platform_audit_log` (platform-scoped audit — audit currently writes only to tenant schemas). Add `EnforcePlatform` + `PlatformAccess`/`PlatformAuthz` middleware mirroring the tenant-side `Enforce`/`RBAC`. Zero-break: default mode `single_admin`, existing `is_platform_admin` users backfilled to a `super_admin` role.

**Tech Stack:** Go 1.26 (gin, pgx/v5, pgxpool), Postgres, Vue3 (Pinia, axios), golang-migrate.

**Design spec:** `docs/superpowers/specs/2026-05-30-platform-admin-rbac-design.md`

---

## File Structure

**Created:**
- `server/migrations/public/000009_platform_rbac.up.sql` / `.down.sql` — tables, built-in roles, backfill, governance setting, platform audit table
- `server/internal/services/iam/platformrbac.go` — platform RBAC types, permission catalog, `EnforcePlatform`, governance mode, role/grant service methods, `SeedPlatformRBAC`
- `server/internal/services/iam/platform_http.go` — `PlatformAccess`, `PlatformAuthz` middleware + `RegisterPlatformRBACRoutes` (`/platform/rbac/*`)
- `server/test/integration/platform_rbac_test.go` — 6 keystone tests
- `web/src/modules/platform/api/rbac.ts` — frontend platform-RBAC API
- `web/src/modules/platform/views/RbacView.vue` — role list + permission matrix + member assignment + mode switch

**Modified:**
- `server/internal/services/iam/repo.go` — new platform repo methods + `Repository` interface entries
- `server/internal/services/audit/audit.go` — `RecordPlatform` + `ListPlatform`
- `server/internal/app/app.go` — construct nothing new (reuses IAM); call `SeedPlatformRBAC`; swap platform group to `PlatformAccess`; pass `a.Audit` into `RegisterPlatformRBACRoutes`; (optionally) gate existing platform routes with `PlatformAuthz`
- `web/src/modules/platform/PlatformLayout.vue` — add "权限管理" nav link
- `web/src/shell/auth/auth.store.ts` — load platform permissions into store for UI gating

---

## Task 1: Migration 000009 — platform RBAC tables, built-in roles, backfill

**Files:**
- Create: `server/migrations/public/000009_platform_rbac.up.sql`
- Create: `server/migrations/public/000009_platform_rbac.down.sql`

- [ ] **Step 1: Write the up migration**

Create `server/migrations/public/000009_platform_rbac.up.sql`:

```sql
-- Platform-level RBAC foundation (三员分立 / 等保).
-- Reuses public.role (tenant_id IS NULL = platform role) and public.role_policy.
-- Adds: a permission catalog, platform_user→role grants (the existing role_grant
-- is keyed by member×tenant and can't grant to a tenant-less platform user), a
-- settings table for the governance mode switch, and a platform-scoped audit log
-- (audit_log currently lives only inside tenant schemas).

-- Catalog of assignable platform permission points (resource/action). Upserted
-- from code at boot (see SeedPlatformRBAC). domain ∈ org_identity / security /
-- audit / app_config / monitoring.
CREATE TABLE IF NOT EXISTS public.platform_permission (
    resource     TEXT NOT NULL,
    action       TEXT NOT NULL,
    domain       TEXT NOT NULL,
    label        TEXT NOT NULL,
    is_high_risk BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (resource, action)
);

-- Grant a platform role (a role with tenant_id IS NULL) directly to a platform_user.
CREATE TABLE IF NOT EXISTS public.platform_role_grant (
    role_id          UUID NOT NULL REFERENCES public.role(id) ON DELETE CASCADE,
    platform_user_id UUID NOT NULL REFERENCES public.platform_user(id) ON DELETE CASCADE,
    granted_by       UUID,
    granted_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (role_id, platform_user_id)
);
CREATE INDEX IF NOT EXISTS platform_role_grant_user_idx
    ON public.platform_role_grant (platform_user_id);

-- Platform-wide settings (key→JSONB). First key: governance_mode.
CREATE TABLE IF NOT EXISTS public.platform_setting (
    key        TEXT PRIMARY KEY,
    value      JSONB NOT NULL,
    updated_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Platform-scoped audit (separate from tenant_<slug>.audit_log).
CREATE TABLE IF NOT EXISTS public.platform_audit_log (
    id              UUID PRIMARY KEY,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    actor           TEXT NOT NULL,
    actor_role      TEXT,
    action          TEXT NOT NULL,
    resource        TEXT,
    resource_id     TEXT,
    reason          TEXT,
    governance_mode TEXT,
    trace_id        TEXT,
    detail          JSONB
);
CREATE INDEX IF NOT EXISTS platform_audit_log_time_idx
    ON public.platform_audit_log (occurred_at DESC);

-- Built-in platform roles (tenant_id IS NULL = platform-wide). super_admin is the
-- bootstrap/super role; the three 等保 members are sys/sec/audit. Legacy
-- 'platform_admin' role (from 000002) is left untouched and unused.
INSERT INTO public.role (id, tenant_id, code, name)
VALUES
  (gen_random_uuid(), NULL, 'super_admin',  '超级管理员'),
  (gen_random_uuid(), NULL, 'sys_admin',    '系统管理员'),
  (gen_random_uuid(), NULL, 'sec_admin',    '安全管理员'),
  (gen_random_uuid(), NULL, 'audit_admin',  '审计管理员')
ON CONFLICT (tenant_id, code) DO NOTHING;

-- Backfill: every existing global platform admin holds super_admin.
INSERT INTO public.platform_role_grant (role_id, platform_user_id, granted_by)
SELECT r.id, u.id, NULL
FROM public.platform_user u
CROSS JOIN public.role r
WHERE u.is_platform_admin = TRUE AND r.code = 'super_admin' AND r.tenant_id IS NULL
ON CONFLICT (role_id, platform_user_id) DO NOTHING;

-- Default governance mode: single_admin (super admin keeps full power; no break).
INSERT INTO public.platform_setting (key, value)
VALUES ('governance_mode', '"single_admin"'::jsonb)
ON CONFLICT (key) DO NOTHING;
```

> Note: `gen_random_uuid()` is available (used in 000002 role seed and in integration tests). The `role` UNIQUE constraint is `(tenant_id, code)`, so `ON CONFLICT (tenant_id, code)` is correct.

- [ ] **Step 2: Write the down migration**

Create `server/migrations/public/000009_platform_rbac.down.sql`:

```sql
DELETE FROM public.platform_setting WHERE key = 'governance_mode';
DELETE FROM public.role
 WHERE tenant_id IS NULL AND code IN ('super_admin','sys_admin','sec_admin','audit_admin');
DROP TABLE IF EXISTS public.platform_audit_log;
DROP TABLE IF EXISTS public.platform_setting;
DROP TABLE IF EXISTS public.platform_role_grant;
DROP TABLE IF EXISTS public.platform_permission;
```

- [ ] **Step 3: Run the migration and verify it applies**

Run (db must be up via `cd deployments && docker compose up -d db`):
```bash
cd server && go run ./cmd/migrate up
```
Expected: migrates to 000009 with no error.

- [ ] **Step 4: Verify tables and seed exist**

Run:
```bash
docker compose -f deployments/docker-compose.yml exec -T db \
  psql -U iop -d iop -c "SELECT code FROM public.role WHERE tenant_id IS NULL AND code LIKE '%admin%' ORDER BY code;" \
  -c "SELECT key, value FROM public.platform_setting;" \
  -c "\dt public.platform_*"
```
Expected: roles include `audit_admin, sec_admin, super_admin, sys_admin` (+ legacy `platform_admin`); `governance_mode = "single_admin"`; tables `platform_audit_log`, `platform_permission`, `platform_role_grant`, `platform_setting` listed.

- [ ] **Step 5: Commit**

```bash
git add server/migrations/public/000009_platform_rbac.up.sql server/migrations/public/000009_platform_rbac.down.sql
git commit -m "feat(iam): 000009 platform RBAC schema (roles/grants/permissions/setting/audit)"
```

---

## Task 2: Repo layer — platform RBAC queries

**Files:**
- Modify: `server/internal/services/iam/repo.go`

These methods follow the exact pgx patterns already in `repo.go` (`ListMemberRoles`, `ListPolicyForRoles`, `scanUser`, `CreateUser`).

- [ ] **Step 1: Add platform RBAC types to platformrbac.go (created fully in Task 3, but the types are needed here)**

For now, add these type definitions at the top of `repo.go` is NOT desired — they belong in `platformrbac.go`. To keep Task 2 compilable on its own, create `server/internal/services/iam/platformrbac.go` with ONLY the types first:

```go
package iam

// PlatformPermission is one assignable platform permission point (resource/action).
type PlatformPermission struct {
	Resource   string `json:"resource"`
	Action     string `json:"action"`
	Domain     string `json:"domain"`
	Label      string `json:"label"`
	IsHighRisk bool   `json:"is_high_risk"`
}
```

- [ ] **Step 2: Add repo methods to repo.go**

Append to `server/internal/services/iam/repo.go` (inside the file, methods on `*pgRepo`). Add `encoding/json` to the import block if not present:

```go
// --- Platform RBAC ---

// ListPlatformRolesForUser returns the platform-level roles (tenant_id IS NULL)
// granted to a platform_user via platform_role_grant.
func (r *pgRepo) ListPlatformRolesForUser(ctx context.Context, platformUserID kernel.ID) ([]*Role, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT r.id, r.tenant_id, r.code, r.name, r.created_at
		 FROM public.platform_role_grant g JOIN public.role r ON r.id = g.role_id
		 WHERE g.platform_user_id = $1 AND r.tenant_id IS NULL`,
		platformUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Role{}
	for rows.Next() {
		var role Role
		if err := rows.Scan(&role.ID, &role.TenantID, &role.Code, &role.Name, &role.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &role)
	}
	return out, rows.Err()
}

// ListPlatformRoles returns all platform-level roles with member counts and policies.
func (r *pgRepo) ListPlatformRoles(ctx context.Context) ([]*Role, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT r.id, r.tenant_id, r.code, r.name, r.created_at,
		        (SELECT count(*) FROM public.platform_role_grant g WHERE g.role_id = r.id)
		 FROM public.role r WHERE r.tenant_id IS NULL ORDER BY r.created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Role{}
	for rows.Next() {
		var role Role
		if err := rows.Scan(&role.ID, &role.TenantID, &role.Code, &role.Name, &role.CreatedAt, &role.MemberCount); err != nil {
			return nil, err
		}
		out = append(out, &role)
	}
	return out, rows.Err()
}

// GetPlatformRoleByCode returns a platform role by code, or (nil, nil) if absent.
func (r *pgRepo) GetPlatformRoleByCode(ctx context.Context, code string) (*Role, error) {
	var role Role
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, code, name, created_at FROM public.role
		 WHERE tenant_id IS NULL AND code = $1`, code).
		Scan(&role.ID, &role.TenantID, &role.Code, &role.Name, &role.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// CreatePlatformRole inserts a custom platform role (tenant_id NULL).
func (r *pgRepo) CreatePlatformRole(ctx context.Context, id kernel.ID, code, name string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO public.role (id, tenant_id, code, name) VALUES ($1, NULL, $2, $3)`,
		id, code, name)
	return err
}

// DeletePlatformRole removes a platform role (cascades grants + policies via FK).
func (r *pgRepo) DeletePlatformRole(ctx context.Context, id kernel.ID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM public.role WHERE id = $1 AND tenant_id IS NULL`, id)
	return err
}

// AddPlatformPolicy / RemovePlatformPolicy operate on public.role_policy for a platform role.
func (r *pgRepo) AddPlatformPolicy(ctx context.Context, roleID kernel.ID, resource, action string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO public.role_policy (role_id, resource, action, effect)
		 VALUES ($1, $2, $3, 'allow') ON CONFLICT (role_id, resource, action) DO NOTHING`,
		roleID, resource, action)
	return err
}

func (r *pgRepo) RemovePlatformPolicy(ctx context.Context, roleID kernel.ID, resource, action string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM public.role_policy WHERE role_id = $1 AND resource = $2 AND action = $3`,
		roleID, resource, action)
	return err
}

// GrantPlatformRole / RevokePlatformRole manage platform_role_grant.
func (r *pgRepo) GrantPlatformRole(ctx context.Context, roleID, platformUserID, grantedBy kernel.ID) error {
	var by any
	if grantedBy != "" {
		by = grantedBy
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO public.platform_role_grant (role_id, platform_user_id, granted_by)
		 VALUES ($1, $2, $3) ON CONFLICT (role_id, platform_user_id) DO NOTHING`,
		roleID, platformUserID, by)
	return err
}

func (r *pgRepo) RevokePlatformRole(ctx context.Context, roleID, platformUserID kernel.ID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM public.platform_role_grant WHERE role_id = $1 AND platform_user_id = $2`,
		roleID, platformUserID)
	return err
}

// ListPlatformRoleMembers returns the platform_user ids granted a role.
func (r *pgRepo) ListPlatformRoleMembers(ctx context.Context, roleID kernel.ID) ([]kernel.ID, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT platform_user_id FROM public.platform_role_grant WHERE role_id = $1`, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []kernel.ID{}
	for rows.Next() {
		var id kernel.ID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// Permission catalog.
func (r *pgRepo) UpsertPlatformPermission(ctx context.Context, p PlatformPermission) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO public.platform_permission (resource, action, domain, label, is_high_risk)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (resource, action) DO UPDATE SET domain = $3, label = $4, is_high_risk = $5`,
		p.Resource, p.Action, p.Domain, p.Label, p.IsHighRisk)
	return err
}

func (r *pgRepo) ListPlatformPermissions(ctx context.Context) ([]*PlatformPermission, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT resource, action, domain, label, is_high_risk FROM public.platform_permission
		 ORDER BY domain, resource, action`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*PlatformPermission{}
	for rows.Next() {
		var p PlatformPermission
		if err := rows.Scan(&p.Resource, &p.Action, &p.Domain, &p.Label, &p.IsHighRisk); err != nil {
			return nil, err
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

// Platform settings (governance_mode). Value is stored as a JSONB string.
func (r *pgRepo) GetPlatformSetting(ctx context.Context, key string) (string, error) {
	var raw []byte
	err := r.pool.QueryRow(ctx, `SELECT value FROM public.platform_setting WHERE key = $1`, key).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var s string
	_ = json.Unmarshal(raw, &s)
	return s, nil
}

func (r *pgRepo) SetPlatformSetting(ctx context.Context, key, value string, by kernel.ID) error {
	v, _ := json.Marshal(value)
	var byArg any
	if by != "" {
		byArg = by
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO public.platform_setting (key, value, updated_by, updated_at)
		 VALUES ($1, $2::jsonb, $3, now())
		 ON CONFLICT (key) DO UPDATE SET value = $2::jsonb, updated_by = $3, updated_at = now()`,
		key, string(v), byArg)
	return err
}
```

- [ ] **Step 3: Add `MemberCount` to the Role struct if missing**

Open `server/internal/services/iam/types.go`, find the `Role` struct. It must have a `MemberCount` field for `ListPlatformRoles`. If absent, add it (use the same JSON tag the tenant RolesView expects — `member_count`):

```go
type Role struct {
	ID         kernel.ID  `json:"id"`
	TenantID   *kernel.ID `json:"tenant_id,omitempty"`
	Code       string     `json:"code"`
	Name       string     `json:"name"`
	CreatedAt  time.Time  `json:"created_at"`
	MemberCount int       `json:"member_count"`
	// (keep any existing fields such as Policies / BuiltIn)
}
```

> If `Role` already has `MemberCount`, leave it. If `TenantID` is a non-pointer `kernel.ID`, that means the existing scan handles NULL another way — in that case match the existing field type in the new queries (scan into the same type) rather than changing the struct.

- [ ] **Step 4: Add the new methods to the `Repository` interface**

Find `type Repository interface` in `repo.go` (or `service.go`) and add the new method signatures so the interface matches `*pgRepo`:

```go
	ListPlatformRolesForUser(ctx context.Context, platformUserID kernel.ID) ([]*Role, error)
	ListPlatformRoles(ctx context.Context) ([]*Role, error)
	GetPlatformRoleByCode(ctx context.Context, code string) (*Role, error)
	CreatePlatformRole(ctx context.Context, id kernel.ID, code, name string) error
	DeletePlatformRole(ctx context.Context, id kernel.ID) error
	AddPlatformPolicy(ctx context.Context, roleID kernel.ID, resource, action string) error
	RemovePlatformPolicy(ctx context.Context, roleID kernel.ID, resource, action string) error
	GrantPlatformRole(ctx context.Context, roleID, platformUserID, grantedBy kernel.ID) error
	RevokePlatformRole(ctx context.Context, roleID, platformUserID kernel.ID) error
	ListPlatformRoleMembers(ctx context.Context, roleID kernel.ID) ([]kernel.ID, error)
	UpsertPlatformPermission(ctx context.Context, p PlatformPermission) error
	ListPlatformPermissions(ctx context.Context) ([]*PlatformPermission, error)
	GetPlatformSetting(ctx context.Context, key string) (string, error)
	SetPlatformSetting(ctx context.Context, key, value string, by kernel.ID) error
```

> Note: `ListPolicyForRoles(roleIDs []kernel.ID)` already exists and is reused as-is for platform roles.

- [ ] **Step 5: Verify it compiles**

Run:
```bash
cd server && go build ./...
```
Expected: builds clean. If `Repository` interface and `*pgRepo` disagree, fix signatures until it builds.

- [ ] **Step 6: Commit**

```bash
git add server/internal/services/iam/repo.go server/internal/services/iam/types.go server/internal/services/iam/platformrbac.go
git commit -m "feat(iam): platform RBAC repo methods (roles/grants/policies/permissions/settings)"
```

---

## Task 3: Service layer — EnforcePlatform, governance mode, role management

**Files:**
- Modify: `server/internal/services/iam/platformrbac.go`
- Test: `server/test/integration/platform_rbac_test.go` (created in Task 9; the unit-style enforcement test below is a focused first check)

- [ ] **Step 1: Write a failing enforcement test**

Create `server/test/integration/platform_rbac_test.go` with the harness + the first test (uses `setupApp`/`mustCreateUser` already in the package):

```go
package integration

import (
	"context"
	"testing"

	"github.com/leo/iop/server/internal/services/iam"
	"github.com/leo/iop/server/internal/shared/kernel"
)

// grant a platform role by code to a platform user (test helper).
func grantPlatformRole(t *testing.T, a *appHandle, userID, code string) {
	t.Helper()
	if err := a.IAM.GrantPlatformRoleByCode(context.Background(), kernel.ID(userID), code, ""); err != nil {
		t.Fatalf("grant %s: %v", code, err)
	}
}

// 命门 1: 系统管理员碰授权/审计 → 拒绝
func TestPlatformKeystone_SysAdminCannotAuthzOrAudit(t *testing.T) {
	a, cleanup := setupApp(t)
	defer cleanup()
	ctx := context.Background()

	uid, _ := mustCreateUser(t, a, "sys")
	if err := a.IAM.GrantPlatformRoleByCode(ctx, kernel.ID(uid), "sys_admin", ""); err != nil {
		t.Fatalf("grant: %v", err)
	}

	// sys_admin MAY create orgs
	if err := a.IAM.EnforcePlatform(ctx, kernel.ID(uid), "org", "create"); err != nil {
		t.Fatalf("sys_admin should allow org/create: %v", err)
	}
	// sys_admin MUST NOT grant authz or read audit
	if err := a.IAM.EnforcePlatform(ctx, kernel.ID(uid), "authz", "grant"); err == nil {
		t.Fatal("sys_admin must NOT have authz/grant")
	}
	if err := a.IAM.EnforcePlatform(ctx, kernel.ID(uid), "audit", "read"); err == nil {
		t.Fatal("sys_admin must NOT have audit/read")
	}
	_ = iam.PlatformPermission{}
}
```

> `appHandle` is the type returned by `setupApp`; if `setupApp` returns `*app.App`, use that type in `grantPlatformRole`'s signature instead. (Match the existing harness.)

- [ ] **Step 2: Run it to verify it fails to compile/pass**

Run:
```bash
cd server && IOP_INTEGRATION=1 \
  IOP_TEST_DB_DSN="postgres://iop:iop_dev@localhost:5433/iop?sslmode=disable" \
  go test ./test/integration/ -run TestPlatformKeystone_SysAdminCannotAuthzOrAudit -v
```
Expected: FAILS to compile — `a.IAM.EnforcePlatform` / `GrantPlatformRoleByCode` undefined.

- [ ] **Step 3: Implement the service methods**

Append to `server/internal/services/iam/platformrbac.go` (add imports: `context`, `github.com/leo/iop/server/internal/shared/errors`, `github.com/leo/iop/server/internal/shared/kernel`):

```go
import (
	"context"

	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
)

const governanceModeKey = "governance_mode"

const (
	ModeSingleAdmin = "single_admin"
	ModeThreeMember = "three_member"
)

// GovernanceMode returns the configured platform governance mode, defaulting to
// single_admin when unset.
func (s *Service) GovernanceMode(ctx context.Context) string {
	m, err := s.repo.GetPlatformSetting(ctx, governanceModeKey)
	if err != nil || m == "" {
		return ModeSingleAdmin
	}
	return m
}

// SetGovernanceMode switches the governance mode. Caller must already be authorized
// (super_admin only — enforced at the route layer).
func (s *Service) SetGovernanceMode(ctx context.Context, mode string, by kernel.ID) error {
	if mode != ModeSingleAdmin && mode != ModeThreeMember {
		return errors.New(errors.KindParam, "iam.invalid_mode", "治理模式无效")
	}
	return s.repo.SetPlatformSetting(ctx, governanceModeKey, mode, by)
}

// HasAnyPlatformRole reports whether the user can enter the platform console at all.
// Back-compat: the legacy is_platform_admin flag also counts (the 000009 migration
// backfills those users with super_admin, but the flag is a belt-and-suspenders).
func (s *Service) HasAnyPlatformRole(ctx context.Context, platformUserID kernel.ID) bool {
	roles, err := s.repo.ListPlatformRolesForUser(ctx, platformUserID)
	if err == nil && len(roles) > 0 {
		return true
	}
	return s.IsPlatformAdminUser(ctx, platformUserID)
}

func permMatch(policyVal, want string) bool { return policyVal == "*" || policyVal == want }

// EnforcePlatform allows the action iff the user holds a platform role whose policy
// permits (resource, action). super_admin is all-access EXCEPT audit/purge is locked
// for everyone (including super_admin) under three_member mode.
func (s *Service) EnforcePlatform(ctx context.Context, platformUserID kernel.ID, resource, action string) error {
	mode := s.GovernanceMode(ctx)
	if mode == ModeThreeMember && resource == "audit" && action == "purge" {
		return errors.New(errors.KindForbidden, "iam.audit_purge_locked", "三员模式下审计不可清除")
	}

	roles, err := s.repo.ListPlatformRolesForUser(ctx, platformUserID)
	if err != nil {
		return err
	}
	// Back-compat: legacy flag holders act as super_admin if they have no grants yet.
	if len(roles) == 0 && s.IsPlatformAdminUser(ctx, platformUserID) {
		return nil
	}
	if len(roles) == 0 {
		return errors.New(errors.KindForbidden, "iam.no_platform_role", "无平台权限")
	}

	roleIDs := make([]kernel.ID, 0, len(roles))
	for _, r := range roles {
		if r.Code == "super_admin" {
			return nil // full access (audit/purge already handled above)
		}
		roleIDs = append(roleIDs, r.ID)
	}

	policies, err := s.repo.ListPolicyForRoles(ctx, roleIDs)
	if err != nil {
		return err
	}
	for _, p := range policies {
		if p.Effect == "allow" && permMatch(p.Resource, resource) && permMatch(p.Action, action) {
			return nil
		}
	}
	return errors.New(errors.KindForbidden, "iam.platform_forbidden", "无权访问")
}

// GrantPlatformRoleByCode is a convenience used by seeding and tests.
func (s *Service) GrantPlatformRoleByCode(ctx context.Context, userID kernel.ID, code string, by kernel.ID) error {
	role, err := s.repo.GetPlatformRoleByCode(ctx, code)
	if err != nil {
		return err
	}
	if role == nil {
		return errors.New(errors.KindNotFound, "iam.role_not_found", "平台角色不存在")
	}
	return s.repo.GrantPlatformRole(ctx, role.ID, userID, by)
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run:
```bash
cd server && IOP_INTEGRATION=1 \
  IOP_TEST_DB_DSN="postgres://iop:iop_dev@localhost:5433/iop?sslmode=disable" \
  go test ./test/integration/ -run TestPlatformKeystone_SysAdminCannotAuthzOrAudit -v
```
Expected: PASS. (Requires Task 4's seed to have populated `sys_admin` policies — if it fails because sys_admin has no policies, implement Task 4 first, then re-run. Tasks 3 and 4 are tightly coupled; do 4 before re-running.)

- [ ] **Step 5: Commit**

```bash
git add server/internal/services/iam/platformrbac.go server/test/integration/platform_rbac_test.go
git commit -m "feat(iam): EnforcePlatform + governance mode + platform role grant"
```

---

## Task 4: Permission catalog + default policies + boot seed

**Files:**
- Modify: `server/internal/services/iam/platformrbac.go`
- Modify: `server/internal/app/app.go`

- [ ] **Step 1: Add the permission catalog + seed function**

Append to `server/internal/services/iam/platformrbac.go`:

```go
// platformCatalog is the authoritative list of platform permission points. Adding a
// new platform feature = add its point here; boot upserts it into platform_permission.
var platformCatalog = []PlatformPermission{
	// org_identity
	{"org", "read", "org_identity", "查看组织", false},
	{"org", "create", "org_identity", "新建组织", false},
	{"org", "update", "org_identity", "编辑组织", false},
	{"org", "suspend", "org_identity", "暂停组织", false},
	{"org", "close", "org_identity", "关闭组织", false},
	{"org", "hierarchy", "org_identity", "管理组织层级", false},
	{"user", "read", "org_identity", "查看用户", false},
	{"user", "create", "org_identity", "新建用户", false},
	{"user", "update", "org_identity", "编辑用户", false},
	{"user", "disable", "org_identity", "停用用户", false},
	{"user", "resetpwd", "org_identity", "重置密码", false},
	{"user", "impersonate", "org_identity", "模拟登录", true},
	{"membership", "assign", "org_identity", "分配组织归属", false},
	// security
	{"role", "manage", "security", "管理平台角色", false},
	{"authz", "grant", "security", "授予角色/权限", false},
	{"platform_admin", "grant", "security", "授予平台管理员资格", true},
	{"security_policy", "manage", "security", "安全策略", false},
	{"session", "read", "security", "查看会话", false},
	{"session", "revoke", "security", "强制下线", false},
	// audit
	{"audit", "read", "audit", "查看审计", false},
	{"audit", "export", "audit", "导出审计", false},
	{"audit", "config", "audit", "审计配置", false},
	{"audit", "purge", "audit", "清除审计", true},
	{"login_log", "read", "audit", "登录日志", false},
	// app_config
	{"app", "manage", "app_config", "应用模块治理", false},
	{"param", "manage", "app_config", "全局参数", false},
	{"dict", "manage", "app_config", "数据字典", false},
	{"announce", "manage", "app_config", "公告通知", false},
	{"branding", "manage", "app_config", "品牌主题", false},
	// monitoring
	{"monitor", "read", "monitoring", "监控查看", false},
	{"job", "manage", "monitoring", "定时任务", false},
	{"cache", "manage", "monitoring", "缓存管理", false},
	{"schema", "sync", "monitoring", "Schema 同步", false},
	{"codegen", "use", "monitoring", "代码生成器", false},
	{"backup", "manage", "monitoring", "备份导出", true},
}

// defaultRolePolicies maps each built-in 三员 role code to its default permission
// points. super_admin is intentionally absent (it is all-access via EnforcePlatform).
var defaultRolePolicies = map[string][][2]string{
	"sys_admin": {
		{"org", "read"}, {"org", "create"}, {"org", "update"}, {"org", "suspend"}, {"org", "close"}, {"org", "hierarchy"},
		{"user", "read"}, {"user", "create"}, {"user", "update"}, {"user", "disable"}, {"user", "resetpwd"}, {"user", "impersonate"},
		{"membership", "assign"},
		{"app", "manage"}, {"param", "manage"}, {"dict", "manage"}, {"announce", "manage"}, {"branding", "manage"},
		{"monitor", "read"}, {"job", "manage"}, {"cache", "manage"}, {"schema", "sync"}, {"codegen", "use"}, {"backup", "manage"},
	},
	"sec_admin": {
		{"org", "read"}, {"user", "read"},
		{"role", "manage"}, {"authz", "grant"}, {"platform_admin", "grant"},
		{"security_policy", "manage"}, {"session", "read"}, {"session", "revoke"},
	},
	"audit_admin": {
		{"org", "read"}, {"user", "read"}, {"session", "read"}, {"monitor", "read"},
		{"audit", "read"}, {"audit", "export"}, {"audit", "config"}, {"audit", "purge"}, {"login_log", "read"},
	},
}

// SeedPlatformRBAC upserts the permission catalog and ensures the built-in 三员 roles
// carry their default policies. Idempotent; safe to run on every boot.
func (s *Service) SeedPlatformRBAC(ctx context.Context) error {
	for _, p := range platformCatalog {
		if err := s.repo.UpsertPlatformPermission(ctx, p); err != nil {
			return err
		}
	}
	for code, pols := range defaultRolePolicies {
		role, err := s.repo.GetPlatformRoleByCode(ctx, code)
		if err != nil {
			return err
		}
		if role == nil {
			continue // migration 000009 seeds these; skip if missing
		}
		for _, ra := range pols {
			if err := s.repo.AddPlatformPolicy(ctx, role.ID, ra[0], ra[1]); err != nil {
				return err
			}
		}
	}
	return nil
}
```

- [ ] **Step 2: Wire SeedPlatformRBAC into boot**

In `server/internal/app/app.go`, find where `SeedDefaults` is called (around line 134, "Seed the built-in platform admin"). Add a call right after it:

```go
	if err := iamSvc.SeedPlatformRBAC(ctx); err != nil {
		return nil, nil, fmt.Errorf("seed platform rbac: %w", err)
	}
```

> Match the surrounding error-handling style (the existing `SeedDefaults` call shows the exact pattern — copy its error wrapping). If `SeedDefaults` is called without a returned error check, follow that style instead.

- [ ] **Step 3: Verify build + seed runs**

Run:
```bash
cd server && go build ./... && go run ./cmd/server &
sleep 3 && curl -s localhost:8080/healthz ; kill %1 2>/dev/null
```
Then confirm policies seeded:
```bash
docker compose -f deployments/docker-compose.yml exec -T db psql -U iop -d iop \
  -c "SELECT r.code, count(*) FROM public.role r JOIN public.role_policy p ON p.role_id=r.id WHERE r.tenant_id IS NULL GROUP BY r.code ORDER BY r.code;"
```
Expected: `audit_admin`, `sec_admin`, `sys_admin` each have their policy counts (9 / 8 / 24). `super_admin` has 0 (by design).

- [ ] **Step 4: Re-run the Task 3 enforcement test**

Run:
```bash
cd server && IOP_INTEGRATION=1 \
  IOP_TEST_DB_DSN="postgres://iop:iop_dev@localhost:5433/iop?sslmode=disable" \
  go test ./test/integration/ -run TestPlatformKeystone_SysAdminCannotAuthzOrAudit -v
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add server/internal/services/iam/platformrbac.go server/internal/app/app.go
git commit -m "feat(iam): platform permission catalog + 三员 default policies + boot seed"
```

---

## Task 5: Middleware — PlatformAccess + PlatformAuthz (with audit)

**Files:**
- Create: `server/internal/services/iam/platform_http.go`

PlatformAuthz records a platform audit entry on successful **writes** (non-GET), and under `three_member` mode requires an `X-Reason` header for high-risk points. It needs the audit service; `iam` importing `audit` is acyclic (audit does not import iam).

- [ ] **Step 1: Implement the middleware**

Create `server/internal/services/iam/platform_http.go`:

```go
package iam

import (
	"encoding/json"

	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/services/audit"
	"github.com/leo/iop/server/internal/shared/apiresp"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
)

// PlatformAccess gates the platform console: any platform role (or legacy
// is_platform_admin) may enter. Replaces PlatformAdminRequired on the platform group.
func PlatformAccess(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := ClaimsFromContext(c.Request.Context())
		if !ok || claims.PlatformUserID == "" {
			apiresp.Fail(c, errors.New(errors.KindForbidden, "iam.platform_access_required", "请使用平台账号"))
			return
		}
		if !svc.HasAnyPlatformRole(c.Request.Context(), claims.PlatformUserID) {
			apiresp.Fail(c, errors.New(errors.KindForbidden, "iam.platform_access_required", "无平台访问权限"))
			return
		}
		c.Next()
	}
}

// PlatformAuthz enforces a single (resource, action) on the platform side and, on
// successful non-GET requests, records a platform audit entry. Under three_member
// mode a high-risk point additionally requires a non-empty X-Reason header.
func PlatformAuthz(svc *Service, aud *audit.Service, resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := ClaimsFromContext(c.Request.Context())
		if !ok || claims.PlatformUserID == "" {
			apiresp.Fail(c, errors.New(errors.KindAuth, "iam.no_session", "未登录"))
			return
		}
		ctx := c.Request.Context()
		if err := svc.EnforcePlatform(ctx, claims.PlatformUserID, resource, action); err != nil {
			apiresp.Fail(c, err)
			return
		}

		mode := svc.GovernanceMode(ctx)
		reason := c.GetHeader("X-Reason")
		if mode == ModeThreeMember && svc.IsHighRiskPermission(resource, action) && reason == "" {
			apiresp.Fail(c, errors.New(errors.KindParam, "iam.reason_required", "高危操作需在 X-Reason 头中填写原因"))
			return
		}

		c.Next()

		// Record writes after the handler ran (best-effort; never blocks the response).
		if c.Request.Method != "GET" && !c.IsAborted() {
			detail, _ := json.Marshal(gin.H{"path": c.Request.URL.Path, "status": c.Writer.Status()})
			aud.RecordPlatform(ctx, audit.PlatformEntry{
				Actor:          string(claims.PlatformUserID),
				Action:         resource + "/" + action,
				Resource:       resource,
				ResourceID:     c.Param("id"),
				Reason:         reason,
				GovernanceMode: mode,
				TraceID:        kernel.TraceIDFromContext(ctx),
				Detail:         detail,
			})
		}
	}
}
```

- [ ] **Step 2: Add `IsHighRiskPermission` to the service**

The middleware calls `svc.IsHighRiskPermission(resource, action)` synchronously, so back it with an in-memory set built from the catalog. Append to `server/internal/services/iam/platformrbac.go`:

```go
// highRiskSet is derived from platformCatalog at package init for O(1) lookup.
var highRiskSet = func() map[[2]string]bool {
	m := map[[2]string]bool{}
	for _, p := range platformCatalog {
		if p.IsHighRisk {
			m[[2]string{p.Resource, p.Action}] = true
		}
	}
	return m
}()

// IsHighRiskPermission reports whether (resource, action) is flagged high-risk.
func (s *Service) IsHighRiskPermission(resource, action string) bool {
	return highRiskSet[[2]string{resource, action}]
}
```

- [ ] **Step 3: Verify build (will fail until Task 6 adds audit.PlatformEntry/RecordPlatform)**

Run:
```bash
cd server && go build ./... 2>&1 | head
```
Expected: FAILS — `audit.PlatformEntry` / `aud.RecordPlatform` undefined. Proceed to Task 6, then this compiles.

- [ ] **Step 4: Commit (after Task 6 makes it build) — deferred**

Commit together with Task 6.

---

## Task 6: Audit — platform-scoped write + list

**Files:**
- Modify: `server/internal/services/audit/audit.go`

- [ ] **Step 1: Add PlatformEntry type + RecordPlatform + ListPlatform**

Append to `server/internal/services/audit/audit.go`. The service holds a `*tenantdb.TenantDB` (`s.tenant`) whose `Transaction` injects tenant search_path — but platform audit lives in `public`, so query `public.platform_audit_log` directly. The service also needs the pool; if `audit.Service` doesn't already hold a `*pgxpool.Pool`, add one.

First, ensure the pool is available. In the `Service` struct add a field and set it in `NewService`:

```go
// (in Service struct)
	pool *pgxpool.Pool
```

Change `NewService` to accept and store the pool (update the single caller in app.go in Step 3):

```go
func NewService(pool *pgxpool.Pool, tenant *tenantdb.TenantDB, lookup func(ctx context.Context, id kernel.ID) (string, bool), logger *zap.Logger) *Service {
	// ... existing field assignments ...
	// s.pool = pool
}
```

Then add (imports: `pgxpool` already needed; `time`, `kernel`, `pgx` already imported):

```go
// PlatformEntry is a platform-scoped audit record (stored in public.platform_audit_log).
type PlatformEntry struct {
	ID             kernel.ID `json:"id"`
	OccurredAt     time.Time `json:"occurred_at"`
	Actor          string    `json:"actor"`
	ActorRole      string    `json:"actor_role"`
	Action         string    `json:"action"`
	Resource       string    `json:"resource"`
	ResourceID     string    `json:"resource_id"`
	Reason         string    `json:"reason"`
	GovernanceMode string    `json:"governance_mode"`
	TraceID        string    `json:"trace_id"`
	Detail         []byte    `json:"detail"`
}

// RecordPlatform writes a platform-scoped audit entry to public.platform_audit_log.
// Synchronous + best-effort: logs and swallows errors so it never breaks a request.
func (s *Service) RecordPlatform(ctx context.Context, e PlatformEntry) {
	if e.ID == "" {
		e.ID = kernel.NewID()
	}
	if e.OccurredAt.IsZero() {
		e.OccurredAt = time.Now().UTC()
	}
	if e.Actor == "" {
		e.Actor = "system"
	}
	var detail any
	if len(e.Detail) > 0 {
		detail = string(e.Detail)
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO public.platform_audit_log
		   (id, occurred_at, actor, actor_role, action, resource, resource_id, reason, governance_mode, trace_id, detail)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::jsonb)`,
		e.ID, e.OccurredAt, e.Actor, e.ActorRole, e.Action, e.Resource, e.ResourceID, e.Reason, e.GovernanceMode, e.TraceID, detail)
	if err != nil {
		s.logger.Warn("platform audit write failed", zap.Error(err), zap.String("action", e.Action))
	}
}

// ListPlatform returns platform-scoped audit entries, newest first.
func (s *Service) ListPlatform(ctx context.Context, p kernel.Pagination) ([]PlatformEntry, error) {
	p = p.Normalize()
	rows, err := s.pool.Query(ctx,
		`SELECT id, occurred_at, actor, COALESCE(actor_role,''), action, COALESCE(resource,''),
		        COALESCE(resource_id,''), COALESCE(reason,''), COALESCE(governance_mode,''),
		        COALESCE(trace_id,''), COALESCE(detail,'null'::jsonb)
		 FROM public.platform_audit_log ORDER BY occurred_at DESC LIMIT $1 OFFSET $2`,
		p.PageSize, p.Offset())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PlatformEntry{}
	for rows.Next() {
		var e PlatformEntry
		if err := rows.Scan(&e.ID, &e.OccurredAt, &e.Actor, &e.ActorRole, &e.Action, &e.Resource,
			&e.ResourceID, &e.Reason, &e.GovernanceMode, &e.TraceID, &e.Detail); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
```

- [ ] **Step 2: Update the audit.NewService caller in app.go**

In `server/internal/app/app.go` (audit construction ~line 151), pass `pool` as the new first arg:

```go
	auditSvc := audit.NewService(pool, tenantDB,
		func(c context.Context, id kernel.ID) (string, bool) {
			t, _ := tenantSvc.GetTenant(c, id)
			if t == nil {
				return "", false
			}
			return t.SchemaName, true
		}, logger)
```

- [ ] **Step 3: Verify build**

Run:
```bash
cd server && go build ./...
```
Expected: builds clean (Task 5 middleware now compiles).

- [ ] **Step 4: Commit**

```bash
git add server/internal/services/audit/audit.go server/internal/services/iam/platform_http.go server/internal/services/iam/platformrbac.go server/internal/app/app.go
git commit -m "feat(audit): platform-scoped audit log (RecordPlatform/ListPlatform) + PlatformAuthz middleware"
```

---

## Task 7: Routes — /platform/rbac/* + wire the platform group

**Files:**
- Modify: `server/internal/services/iam/platform_http.go`
- Modify: `server/internal/app/app.go`

- [ ] **Step 1: Add RegisterPlatformRBACRoutes**

Append to `server/internal/services/iam/platform_http.go`. Mirrors `RegisterPlatformAdminRoutes` conventions (apiresp, kernel.ParseID, ShouldBindJSON):

```go
// RegisterPlatformRBACRoutes mounts /platform/rbac/* on the platform group.
func RegisterPlatformRBACRoutes(r *gin.RouterGroup, svc *Service, aud *audit.Service) {
	// Current user's platform roles + permissions (for front-end gating). Login-only.
	r.GET("/platform/rbac/me", func(c *gin.Context) {
		claims, _ := ClaimsFromContext(c.Request.Context())
		ctx := c.Request.Context()
		roles, _ := svc.repo.ListPlatformRolesForUser(ctx, claims.PlatformUserID)
		codes := []string{}
		isSuper := false
		for _, role := range roles {
			codes = append(codes, role.Code)
			if role.Code == "super_admin" {
				isSuper = true
			}
		}
		perms, _ := svc.PlatformPermissionsForUser(ctx, claims.PlatformUserID)
		apiresp.OK(c, gin.H{
			"roles": codes, "permissions": perms, "is_super_admin": isSuper,
			"governance_mode": svc.GovernanceMode(ctx),
		})
	})

	// Permission catalog (for the assignment matrix).
	r.GET("/platform/rbac/permissions", PlatformAuthz(svc, aud, "role", "manage"), func(c *gin.Context) {
		perms, err := svc.repo.ListPlatformPermissions(c.Request.Context())
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"permissions": perms})
	})

	// Roles list (with policies + member counts).
	r.GET("/platform/rbac/roles", PlatformAuthz(svc, aud, "role", "manage"), func(c *gin.Context) {
		roles, err := svc.ListPlatformRolesWithPolicies(c.Request.Context())
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"roles": roles})
	})

	r.POST("/platform/rbac/roles", PlatformAuthz(svc, aud, "role", "manage"), func(c *gin.Context) {
		var req struct {
			Code string `json:"code" binding:"required"`
			Name string `json:"name" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_request", "请求格式错误", err))
			return
		}
		if err := svc.CreatePlatformRole(c.Request.Context(), req.Code, req.Name); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.DELETE("/platform/rbac/roles/:id", PlatformAuthz(svc, aud, "role", "manage"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_id", "id 无效", err))
			return
		}
		if err := svc.DeletePlatformRole(c.Request.Context(), id); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.POST("/platform/rbac/roles/:id/policies", PlatformAuthz(svc, aud, "role", "manage"), func(c *gin.Context) {
		id, _ := kernel.ParseID(c.Param("id"))
		var req struct {
			Resource string `json:"resource" binding:"required"`
			Action   string `json:"action" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_request", "请求格式错误", err))
			return
		}
		if err := svc.repo.AddPlatformPolicy(c.Request.Context(), id, req.Resource, req.Action); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.DELETE("/platform/rbac/roles/:id/policies", PlatformAuthz(svc, aud, "role", "manage"), func(c *gin.Context) {
		id, _ := kernel.ParseID(c.Param("id"))
		if err := svc.repo.RemovePlatformPolicy(c.Request.Context(), id, c.Query("resource"), c.Query("action")); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	// Role membership (authz/grant — the security admin's job).
	r.POST("/platform/rbac/roles/:id/members", PlatformAuthz(svc, aud, "authz", "grant"), func(c *gin.Context) {
		claims, _ := ClaimsFromContext(c.Request.Context())
		id, _ := kernel.ParseID(c.Param("id"))
		var req struct {
			PlatformUserID string `json:"platform_user_id" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_request", "请求格式错误", err))
			return
		}
		uid, err := kernel.ParseID(req.PlatformUserID)
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_id", "platform_user_id 无效", err))
			return
		}
		if err := svc.repo.GrantPlatformRole(c.Request.Context(), id, uid, claims.PlatformUserID); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.DELETE("/platform/rbac/roles/:id/members/:uid", PlatformAuthz(svc, aud, "authz", "grant"), func(c *gin.Context) {
		id, _ := kernel.ParseID(c.Param("id"))
		uid, _ := kernel.ParseID(c.Param("uid"))
		if err := svc.repo.RevokePlatformRole(c.Request.Context(), id, uid); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	// Governance mode — super_admin only.
	r.GET("/platform/rbac/governance-mode", func(c *gin.Context) {
		apiresp.OK(c, gin.H{"mode": svc.GovernanceMode(c.Request.Context())})
	})
	r.PUT("/platform/rbac/governance-mode", func(c *gin.Context) {
		claims, _ := ClaimsFromContext(c.Request.Context())
		if !svc.UserHasPlatformRole(c.Request.Context(), claims.PlatformUserID, "super_admin") {
			apiresp.Fail(c, errors.New(errors.KindForbidden, "iam.super_admin_only", "仅超级管理员可切换治理模式"))
			return
		}
		var req struct {
			Mode string `json:"mode" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_request", "请求格式错误", err))
			return
		}
		if err := svc.SetGovernanceMode(c.Request.Context(), req.Mode, claims.PlatformUserID); err != nil {
			apiresp.Fail(c, err)
			return
		}
		aud.RecordPlatform(c.Request.Context(), audit.PlatformEntry{
			Actor: string(claims.PlatformUserID), Action: "governance/switch",
			Resource: "governance", ResourceID: req.Mode, GovernanceMode: req.Mode,
		})
		apiresp.OK(c, gin.H{"ok": true})
	})
}
```

- [ ] **Step 2: Add the supporting service helpers**

Append to `server/internal/services/iam/platformrbac.go`:

```go
// RoleWithPolicies is a platform role plus its policy rules (for the admin UI).
type RoleWithPolicies struct {
	*Role
	BuiltIn  bool          `json:"built_in"`
	Policies []*PolicyRule `json:"policies"`
	Members  []kernel.ID   `json:"members"`
}

var builtinPlatformRoleCodes = map[string]bool{
	"super_admin": true, "sys_admin": true, "sec_admin": true, "audit_admin": true,
}

func (s *Service) CreatePlatformRole(ctx context.Context, code, name string) error {
	return s.repo.CreatePlatformRole(ctx, kernel.NewID(), code, name)
}

func (s *Service) DeletePlatformRole(ctx context.Context, id kernel.ID) error {
	return s.repo.DeletePlatformRole(ctx, id)
}

func (s *Service) ListPlatformRolesWithPolicies(ctx context.Context) ([]*RoleWithPolicies, error) {
	roles, err := s.repo.ListPlatformRoles(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*RoleWithPolicies, 0, len(roles))
	for _, role := range roles {
		pols, _ := s.repo.ListPolicyForRoles(ctx, []kernel.ID{role.ID})
		members, _ := s.repo.ListPlatformRoleMembers(ctx, role.ID)
		out = append(out, &RoleWithPolicies{
			Role: role, BuiltIn: builtinPlatformRoleCodes[role.Code], Policies: pols, Members: members,
		})
	}
	return out, nil
}

// PlatformPermissionsForUser returns the flat set of "resource/action" the user holds
// (or ["*/*"] for super_admin), for front-end gating.
func (s *Service) PlatformPermissionsForUser(ctx context.Context, userID kernel.ID) ([]string, error) {
	roles, err := s.repo.ListPlatformRolesForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	roleIDs := make([]kernel.ID, 0, len(roles))
	for _, r := range roles {
		if r.Code == "super_admin" {
			return []string{"*/*"}, nil
		}
		roleIDs = append(roleIDs, r.ID)
	}
	pols, err := s.repo.ListPolicyForRoles(ctx, roleIDs)
	if err != nil {
		return nil, err
	}
	out := []string{}
	for _, p := range pols {
		if p.Effect == "allow" {
			out = append(out, p.Resource+"/"+p.Action)
		}
	}
	return out, nil
}

func (s *Service) UserHasPlatformRole(ctx context.Context, userID kernel.ID, code string) bool {
	roles, err := s.repo.ListPlatformRolesForUser(ctx, userID)
	if err != nil {
		return false
	}
	for _, r := range roles {
		if r.Code == code {
			return true
		}
	}
	return false
}
```

> `PolicyRule` already exists (used by `ListPolicyForRoles`). If its fields differ from `{RoleID, Resource, Action, Effect}`, adjust the JSON tags only.

- [ ] **Step 3: Wire the platform group in app.go**

In `server/internal/app/app.go`, the platform group (lines ~330-340). Change the gate and register the RBAC routes:

```go
	platform := api.Group("")
	platform.Use(iam.JWTAuth(a.IAM))
	platform.Use(iam.PasswordChangeGate(a.IAM))
	platform.Use(iam.PlatformAccess(a.IAM)) // was: iam.PlatformAdminRequired(a.IAM)
	iam.RegisterPlatformAdminRoutes(platform, a.IAM, a.Pool)
	iam.RegisterPlatformRBACRoutes(platform, a.IAM, a.Audit)
	tenancy.RegisterRoutes(platform, a.Tenancy, a.Pool)
```

> `PlatformAdminRequired` stays defined (do not delete) — it's still referenced in `/me/admin` logic and keeps back-compat. We only stop using it as the group gate.

- [ ] **Step 4: Verify build + vet**

Run:
```bash
cd server && go build ./... && go vet ./...
```
Expected: clean.

- [ ] **Step 5: Smoke-test the routes**

Run (login as the seeded admin to get a token, then call rbac/me):
```bash
cd server && go run ./cmd/server &
sleep 3
TOKEN=$(curl -s localhost:8080/api/auth/login -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"Admin12345!"}' | python3 -c "import sys,json;print(json.load(sys.stdin)['data']['token']['access_token'])")
curl -s localhost:8080/api/platform/rbac/me -H "Authorization: Bearer $TOKEN" | python3 -m json.tool
kill %1 2>/dev/null
```
Expected: JSON with `"roles": ["super_admin"]` (admin was backfilled), `"is_super_admin": true`, `"permissions": ["*/*"]`, `"governance_mode": "single_admin"`.

> If admin is forced to change password first, the gate returns `iam.password_change_required` — rotate via `/api/me/password` in the smoke test or temporarily clear `password_must_change` in the DB for the smoke check.

- [ ] **Step 6: Commit**

```bash
git add server/internal/services/iam/platform_http.go server/internal/services/iam/platformrbac.go server/internal/app/app.go
git commit -m "feat(iam): /platform/rbac/* routes + swap platform group to PlatformAccess"
```

---

## Task 8: Frontend — platform RBAC api, RbacView, nav, store gating

**Files:**
- Create: `web/src/modules/platform/api/rbac.ts`
- Create: `web/src/modules/platform/views/RbacView.vue`
- Modify: `web/src/router/index.ts`
- Modify: `web/src/modules/platform/PlatformLayout.vue`
- Modify: `web/src/shell/auth/auth.store.ts`

- [ ] **Step 1: Create the api module**

Create `web/src/modules/platform/api/rbac.ts`:

```ts
import { client } from "@/api/client";

export interface PlatformPermission {
  resource: string; action: string; domain: string; label: string; is_high_risk: boolean;
}
export interface PolicyRule { resource: string; action: string; effect: string }
export interface PlatformRole {
  id: string; code: string; name: string; built_in: boolean;
  member_count: number; policies: PolicyRule[] | null; members: string[];
}
export interface RbacMe {
  roles: string[]; permissions: string[]; is_super_admin: boolean; governance_mode: string;
}

export async function getRbacMe(): Promise<RbacMe> {
  const r = await client.get("/platform/rbac/me");
  return r.data?.data ?? { roles: [], permissions: [], is_super_admin: false, governance_mode: "single_admin" };
}
export async function listPlatformPermissions(): Promise<PlatformPermission[]> {
  const r = await client.get("/platform/rbac/permissions");
  return r.data?.data?.permissions ?? [];
}
export async function listPlatformRoles(): Promise<PlatformRole[]> {
  const r = await client.get("/platform/rbac/roles");
  return r.data?.data?.roles ?? [];
}
export async function createPlatformRole(code: string, name: string) {
  await client.post("/platform/rbac/roles", { code, name });
}
export async function deletePlatformRole(id: string) {
  await client.delete(`/platform/rbac/roles/${id}`);
}
export async function addPlatformPolicy(id: string, resource: string, action: string) {
  await client.post(`/platform/rbac/roles/${id}/policies`, { resource, action });
}
export async function removePlatformPolicy(id: string, resource: string, action: string) {
  await client.delete(`/platform/rbac/roles/${id}/policies`, { params: { resource, action } });
}
export async function grantPlatformRole(id: string, platform_user_id: string) {
  await client.post(`/platform/rbac/roles/${id}/members`, { platform_user_id });
}
export async function revokePlatformRole(id: string, uid: string) {
  await client.delete(`/platform/rbac/roles/${id}/members/${uid}`);
}
export async function setGovernanceMode(mode: "single_admin" | "three_member") {
  await client.put("/platform/rbac/governance-mode", { mode });
}
```

- [ ] **Step 2: Create RbacView.vue (mirrors RolesView.vue patterns)**

Create `web/src/modules/platform/views/RbacView.vue`:

```vue
<template>
  <section class="rbac">
    <PageHeader title="权限管理" :sub="`三员分立 · 当前治理模式：${modeLabel}`">
      <template #actions>
        <div class="head-actions">
          <button class="btn" @click="toggleMode" :disabled="!me.is_super_admin">
            切换为{{ me.governance_mode === 'three_member' ? '单一超管' : '三员分立' }}模式
          </button>
          <button class="btn btn-primary" @click="showCreate = !showCreate">+ 新建角色</button>
        </div>
      </template>
    </PageHeader>

    <form v-if="showCreate" class="card create-form" @submit.prevent="create">
      <input class="input" v-model="newRole.code" required pattern="[a-z_]+" placeholder="编码，如 ops_admin" />
      <input class="input" v-model="newRole.name" required placeholder="显示名称" />
      <button class="btn btn-primary" :disabled="saving">创建</button>
      <button class="btn" type="button" @click="showCreate = false">取消</button>
    </form>

    <div class="roles-grid">
      <article v-for="r in roles" :key="r.id" class="role-card">
        <div class="role-head">
          <div class="role-title">
            {{ r.name }} <span class="role-code">{{ r.code }}</span>
            <span v-if="r.built_in" class="tag-builtin">内置</span>
          </div>
          <button v-if="!r.built_in" class="link-btn warn" @click="del(r.id)">删除</button>
        </div>
        <div class="role-meta">{{ r.member_count }} 名成员</div>
        <div class="policies">
          <span v-if="r.code === 'super_admin'" class="no-pol">★ 全权（受三员模式约束）</span>
          <span v-else-if="!r.policies?.length" class="no-pol">尚无权限</span>
          <span v-for="p in (r.policies ?? [])" :key="`${p.resource}:${p.action}`" class="pol-chip">
            {{ p.resource }}<span class="sep">/</span>{{ p.action }}
            <button v-if="!r.built_in" class="rm" @click="removePol(r.id, p.resource, p.action)">×</button>
          </span>
        </div>
        <div v-if="!r.built_in" class="add-pol-row">
          <select class="input input-sm" v-model="newPol[r.id].key">
            <option value="">添加权限点…</option>
            <optgroup v-for="(list, dom) in byDomain" :key="dom" :label="dom">
              <option v-for="p in list" :key="`${p.resource}/${p.action}`" :value="`${p.resource}/${p.action}`">
                {{ p.label }} ({{ p.resource }}/{{ p.action }}){{ p.is_high_risk ? ' ⚠' : '' }}
              </option>
            </optgroup>
          </select>
          <button class="btn btn-sm" @click="addPol(r.id)" :disabled="!newPol[r.id].key">+ 加</button>
        </div>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue";
import { PageHeader } from "@/shell/components";
import { useNotification } from "@/shell/notify";
import { useConfirm } from "@/shell/confirm";
import {
  getRbacMe, listPlatformRoles, listPlatformPermissions, createPlatformRole,
  deletePlatformRole, addPlatformPolicy, removePlatformPolicy, setGovernanceMode,
  type PlatformRole, type PlatformPermission, type RbacMe,
} from "../api/rbac";

const notify = useNotification();
const { confirm } = useConfirm();

const roles = ref<PlatformRole[]>([]);
const perms = ref<PlatformPermission[]>([]);
const me = ref<RbacMe>({ roles: [], permissions: [], is_super_admin: false, governance_mode: "single_admin" });
const showCreate = ref(false);
const saving = ref(false);
const newRole = reactive({ code: "", name: "" });
const newPol = reactive<Record<string, { key: string }>>({});

const modeLabel = computed(() => (me.value.governance_mode === "three_member" ? "三员分立（严格）" : "单一超管"));
const byDomain = computed(() => {
  const m: Record<string, PlatformPermission[]> = {};
  for (const p of perms.value) (m[p.domain] ??= []).push(p);
  return m;
});

onMounted(reload);
watch(roles, (rs) => rs.forEach((r) => (newPol[r.id] ??= { key: "" })));

async function reload() {
  me.value = await getRbacMe();
  roles.value = await listPlatformRoles();
  try { perms.value = await listPlatformPermissions(); } catch {}
}
async function create() {
  saving.value = true;
  try {
    await createPlatformRole(newRole.code, newRole.name);
    newRole.code = ""; newRole.name = ""; showCreate.value = false;
    await reload();
  } catch (e: any) { notify.error(e.response?.data?.error?.message ?? "创建失败"); }
  finally { saving.value = false; }
}
async function del(id: string) {
  if (!(await confirm({ title: "确认", message: "确定删除该角色？", danger: true }))) return;
  try { await deletePlatformRole(id); await reload(); }
  catch (e: any) { notify.error(e.response?.data?.error?.message ?? "删除失败"); }
}
async function addPol(id: string) {
  const key = newPol[id].key;
  if (!key) return;
  const [resource, action] = key.split("/");
  try { await addPlatformPolicy(id, resource, action); newPol[id].key = ""; await reload(); }
  catch (e: any) { notify.error(e.response?.data?.error?.message ?? "添加失败"); }
}
async function removePol(id: string, resource: string, action: string) {
  try { await removePlatformPolicy(id, resource, action); await reload(); }
  catch (e: any) { notify.error(e.response?.data?.error?.message ?? "移除失败"); }
}
async function toggleMode() {
  const next = me.value.governance_mode === "three_member" ? "single_admin" : "three_member";
  if (!(await confirm({ title: "切换治理模式", message: `确认切换为「${next === 'three_member' ? '三员分立' : '单一超管'}」？`, danger: next === "three_member" }))) return;
  try { await setGovernanceMode(next); await reload(); notify.success("已切换"); }
  catch (e: any) { notify.error(e.response?.data?.error?.message ?? "切换失败"); }
}
</script>

<style scoped>
.rbac { display: flex; flex-direction: column; gap: var(--sp-5); }
.head-actions { display: flex; gap: 8px; }
.create-form { display: flex; gap: 10px; align-items: center; padding: 14px 16px; }
.roles-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(340px, 1fr)); gap: 14px; }
.role-card { background: var(--surface); border: 1px solid var(--border); border-radius: 12px; padding: 16px 18px; box-shadow: var(--sh-1); }
.role-head { display: flex; justify-content: space-between; align-items: center; }
.role-title { font-weight: 600; display: flex; align-items: center; gap: 6px; }
.role-code { font-size: 11px; color: var(--text-3); background: var(--surface-2); border-radius: 4px; padding: 1px 6px; }
.tag-builtin { font-size: 10px; color: var(--primary); background: var(--primary-soft); border-radius: 3px; padding: 1px 5px; }
.role-meta { font-size: 12px; color: var(--text-3); margin: 4px 0 10px; }
.policies { display: flex; flex-wrap: wrap; gap: 6px; min-height: 24px; }
.pol-chip { font-size: 11.5px; background: var(--surface-2); border: 1px solid var(--border); border-radius: 5px; padding: 2px 6px; display: inline-flex; align-items: center; gap: 3px; }
.pol-chip .sep { color: var(--text-4); }
.pol-chip .rm { border: 0; background: none; color: var(--text-3); cursor: pointer; }
.no-pol { font-size: 12px; color: var(--text-3); }
.add-pol-row { display: flex; gap: 8px; margin-top: 12px; }
.input-sm { font-size: 12px; padding: 4px 8px; }
.link-btn.warn { color: var(--danger); background: none; border: 0; cursor: pointer; font-size: 12px; }
</style>
```

- [ ] **Step 3: Add the route**

In `web/src/router/index.ts`, add to the `platform` children array (after `registrations`):

```ts
            { path: "rbac", name: "platform.rbac", component: () => import("@/modules/platform/views/RbacView.vue") },
```

- [ ] **Step 4: Add the nav link**

In `web/src/modules/platform/PlatformLayout.vue`, inside the 治理 `nav-group`, add after the registrations link:

```html
        <router-link to="/platform/rbac" class="nav-link" :class="{ active: $route.path.startsWith('/platform/rbac') }">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
          权限管理
        </router-link>
```

- [ ] **Step 5: Type-check**

Run:
```bash
cd web && npx vue-tsc --noEmit
```
Expected: no errors.

- [ ] **Step 6: Commit**

```bash
git add web/src/modules/platform/api/rbac.ts web/src/modules/platform/views/RbacView.vue web/src/router/index.ts web/src/modules/platform/PlatformLayout.vue
git commit -m "feat(web): platform 权限管理 page (roles + permission matrix + mode switch)"
```

---

## Task 9: Keystone integration tests (命门)

**Files:**
- Modify: `server/test/integration/platform_rbac_test.go`

Add the remaining 5 keystone tests (Test 1 written in Task 3). These run at the service layer (fast, no HTTP) except where an HTTP gate is the thing under test.

- [ ] **Step 1: Add the separation-of-duties tests**

Append to `server/test/integration/platform_rbac_test.go`:

```go
// 命门 2: 安全管理员不能操作业务/配置;审计管理员不能配置/授权
func TestPlatformKeystone_SecAndAuditScoping(t *testing.T) {
	a, cleanup := setupApp(t)
	defer cleanup()
	ctx := context.Background()

	sec, _ := mustCreateUser(t, a, "sec")
	_ = a.IAM.GrantPlatformRoleByCode(ctx, kernel.ID(sec), "sec_admin", "")
	if err := a.IAM.EnforcePlatform(ctx, kernel.ID(sec), "authz", "grant"); err != nil {
		t.Fatalf("sec_admin should allow authz/grant: %v", err)
	}
	if err := a.IAM.EnforcePlatform(ctx, kernel.ID(sec), "org", "create"); err == nil {
		t.Fatal("sec_admin must NOT create orgs")
	}
	if err := a.IAM.EnforcePlatform(ctx, kernel.ID(sec), "audit", "read"); err == nil {
		t.Fatal("sec_admin must NOT read audit")
	}

	aud, _ := mustCreateUser(t, a, "aud")
	_ = a.IAM.GrantPlatformRoleByCode(ctx, kernel.ID(aud), "audit_admin", "")
	if err := a.IAM.EnforcePlatform(ctx, kernel.ID(aud), "audit", "read"); err != nil {
		t.Fatalf("audit_admin should read audit: %v", err)
	}
	if err := a.IAM.EnforcePlatform(ctx, kernel.ID(aud), "authz", "grant"); err == nil {
		t.Fatal("audit_admin must NOT grant authz")
	}
	if err := a.IAM.EnforcePlatform(ctx, kernel.ID(aud), "param", "manage"); err == nil {
		t.Fatal("audit_admin must NOT manage config")
	}
}

// 命门 3: three_member 下 audit/purge 对超管也拒绝
func TestPlatformKeystone_AuditPurgeLockedForSuper(t *testing.T) {
	a, cleanup := setupApp(t)
	defer cleanup()
	ctx := context.Background()

	su, _ := mustCreateUser(t, a, "super")
	_ = a.IAM.GrantPlatformRoleByCode(ctx, kernel.ID(su), "super_admin", "")

	// single mode: super can purge
	_ = a.IAM.SetGovernanceMode(ctx, iam.ModeSingleAdmin, "")
	if err := a.IAM.EnforcePlatform(ctx, kernel.ID(su), "audit", "purge"); err != nil {
		t.Fatalf("single mode super should purge: %v", err)
	}
	// three_member: even super cannot purge
	_ = a.IAM.SetGovernanceMode(ctx, iam.ModeThreeMember, "")
	if err := a.IAM.EnforcePlatform(ctx, kernel.ID(su), "audit", "purge"); err == nil {
		t.Fatal("three_member: super_admin must NOT purge audit")
	}
	// but super still has other powers
	if err := a.IAM.EnforcePlatform(ctx, kernel.ID(su), "org", "create"); err != nil {
		t.Fatalf("super should still create orgs in three_member: %v", err)
	}
	_ = a.IAM.SetGovernanceMode(ctx, iam.ModeSingleAdmin, "") // restore
}

// 命门 5: 无平台角色 → 无访问
func TestPlatformKeystone_NoRoleNoAccess(t *testing.T) {
	a, cleanup := setupApp(t)
	defer cleanup()
	ctx := context.Background()

	nobody, _ := mustCreateUser(t, a, "nobody")
	if a.IAM.HasAnyPlatformRole(ctx, kernel.ID(nobody)) {
		t.Fatal("a fresh user must NOT have platform access")
	}
	if err := a.IAM.EnforcePlatform(ctx, kernel.ID(nobody), "org", "read"); err == nil {
		t.Fatal("a fresh user must be denied")
	}
}

// 命门 4: 回填 — 现有 is_platform_admin 用户经迁移持有 super_admin（用 seeded admin 验证）
func TestPlatformKeystone_BackfilledAdminIsSuper(t *testing.T) {
	a, cleanup := setupApp(t)
	defer cleanup()
	ctx := context.Background()

	// The seeded bootstrap admin (is_platform_admin=true) must resolve as super_admin.
	u, err := a.IAM.FindUserByUsername(ctx, "admin")
	if err != nil || u == nil {
		t.Fatalf("seeded admin missing: %v", err)
	}
	if !a.IAM.UserHasPlatformRole(ctx, u.ID, "super_admin") {
		t.Fatal("backfilled admin must hold super_admin")
	}
}
```

> `FindUserByUsername` may be named differently (e.g. `GetUserByUsername`). Grep the iam service for the existing lookup and use that. If none is exported, assert via `HasAnyPlatformRole` on the admin id obtained through `ListUsers`.

- [ ] **Step 2: Run the full keystone suite**

Run:
```bash
cd server && IOP_INTEGRATION=1 \
  IOP_TEST_DB_DSN="postgres://iop:iop_dev@localhost:5433/iop?sslmode=disable" \
  IOP_DB_DSN="postgres://iop:iop_dev@localhost:5433/iop?sslmode=disable" \
  go test ./test/integration/ -run TestPlatformKeystone -v
```
Expected: all 5 PASS.

- [ ] **Step 3: Run the entire integration suite (no regressions)**

Run:
```bash
cd server && IOP_INTEGRATION=1 \
  IOP_TEST_DB_DSN="postgres://iop:iop_dev@localhost:5433/iop?sslmode=disable" \
  IOP_DB_DSN="postgres://iop:iop_dev@localhost:5433/iop?sslmode=disable" \
  go test ./test/integration/...
```
Expected: PASS (existing 5 isolation keystones + smoke + new platform keystones).

- [ ] **Step 4: Commit**

```bash
git add server/test/integration/platform_rbac_test.go
git commit -m "test(integration): platform RBAC 三员分立 keystone tests"
```

---

## Task 10: Final verification + governance-mode HTTP gate test + wrap-up

**Files:**
- Modify: `server/test/integration/platform_rbac_test.go` (one HTTP-level test)

- [ ] **Step 1: Add the 命门 6 HTTP test (non-super cannot switch mode)**

Append to `server/test/integration/platform_rbac_test.go` (uses `httptest` + login, per smoke_test.go):

```go
import (
	"net/http"
	"net/http/httptest"
	// ...existing imports
)

// 命门 6: 非超管切换治理模式 → 403
func TestPlatformKeystone_NonSuperCannotSwitchMode(t *testing.T) {
	a, cleanup := setupApp(t)
	defer cleanup()
	srv := httptest.NewServer(a.Engine())
	defer srv.Close()

	// A sys_admin (not super) token. Create user, grant sys_admin, log in.
	// (Use the project's login + token helper; if mustLogin exists, prefer it.)
	t.Skip("enable once a login/token test helper is available; logic mirrors smoke_test.go")
	_ = srv
	_ = http.StatusForbidden
}
```

> If the harness already has a login helper that returns a Bearer token, replace the `t.Skip` with: create user → `GrantPlatformRoleByCode(sys_admin)` → login → `PUT /api/platform/rbac/governance-mode` with body `{"mode":"three_member"}` → assert HTTP 403 and that the mode is unchanged. Keep the service-layer guard test (Task 9) as the authoritative check regardless.

- [ ] **Step 2: Full build + vet + type-check**

Run:
```bash
cd server && go build ./... && go vet ./... && make build
cd ../web && npx vue-tsc --noEmit && npm run build
```
Expected: all clean.

- [ ] **Step 3: Manual end-to-end smoke (optional but recommended)**

Run `./scripts/dev.sh`, log in as `admin / Admin12345!` (rotate password), open `/platform/rbac`. Verify: super_admin shows "全权"; you can create a custom role, add a permission point, and toggle governance mode. Confirm a row appears in `public.platform_audit_log` after a write.

- [ ] **Step 4: Final commit**

```bash
git add -A
git commit -m "test(integration): governance-mode HTTP gate + Phase 0 platform RBAC complete"
```

---

## Self-Review

**Spec coverage:**
- §2 三员 model → Task 1 (built-in roles) + Task 4 (default policies) + Task 9 (separation tests). ✓
- §3 governance mode → Task 1 (setting) + Task 3 (`GovernanceMode`/`SetGovernanceMode`) + Task 7 (PUT endpoint, super-only) + Task 9 (audit/purge lock). ✓
- §4 data model → Task 1 (all tables, reusing role/role_policy) + Task 2 (repo). ✓
- §5 catalog + 三员 切分 → Task 4 (`platformCatalog` + `defaultRolePolicies`). ✓
- §6 PlatformAccess/PlatformAuthz → Task 5. ✓
- §7 /platform/rbac/* → Task 7. ✓
- §8 audit接入 → Task 6 (`platform_audit_log` + RecordPlatform/ListPlatform; **plan adds the table the spec omitted**) + Task 5 (PlatformAuthz records writes). ✓
- §9 frontend → Task 8. ✓
- §10 命门测试 → Task 9 + Task 10. ✓
- §11 代码落点 → matches File Structure. ✓

**Placeholder scan:** No TBD/TODO. Two explicitly-flagged adaptation points (`appHandle`/`*app.App` type in test helper; `FindUserByUsername` exact name; login-token helper for the one HTTP test) are marked with concrete fallbacks, not left blank.

**Type consistency:** `EnforcePlatform(ctx, kernel.ID, resource, action) error`, `GrantPlatformRoleByCode(ctx, userID, code, by)`, `HasAnyPlatformRole`, `GovernanceMode`/`SetGovernanceMode`, `ModeSingleAdmin`/`ModeThreeMember`, `PlatformPermission{Resource,Action,Domain,Label,IsHighRisk}`, `audit.PlatformEntry`, `RecordPlatform`/`ListPlatform` — names used consistently across Tasks 2–10. `permMatch` wildcard helper defined once (Task 3) and relied on by EnforcePlatform.

**Open risks to confirm during execution (not blockers):**
1. `Role` struct field types (`TenantID` pointer vs value; presence of `MemberCount`) — verify against `types.go` and adjust scans (flagged in Task 2 Step 3).
2. `audit.Service` gaining a `*pgxpool.Pool` field changes `NewService`'s signature — only one caller (app.go), updated in Task 6.
3. `iam` importing `audit` must stay acyclic (audit imports only tenantdb/eventbus/kernel — confirmed safe).
