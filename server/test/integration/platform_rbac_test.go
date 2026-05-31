package integration

import (
	"context"
	"testing"

	"github.com/leo/iop/server/internal/shared/kernel"
)

// 命门 1: 系统管理员碰授权/审计 → 拒绝
func TestPlatformKeystone_SysAdminCannotAuthzOrAudit(t *testing.T) {
	a, cleanup := setupApp(t)
	defer cleanup()
	ctx := context.Background()

	uid, _ := mustCreateUser(t, a, "sys")
	if err := a.IAM.GrantPlatformRoleByCode(ctx, kernel.ID(uid), "sys_admin", ""); err != nil {
		t.Fatalf("grant: %v", err)
	}

	if err := a.IAM.EnforcePlatform(ctx, kernel.ID(uid), "org", "create"); err != nil {
		t.Fatalf("sys_admin should allow org/create: %v", err)
	}
	if err := a.IAM.EnforcePlatform(ctx, kernel.ID(uid), "authz", "grant"); err == nil {
		t.Fatal("sys_admin must NOT have authz/grant")
	}
	if err := a.IAM.EnforcePlatform(ctx, kernel.ID(uid), "audit", "read"); err == nil {
		t.Fatal("sys_admin must NOT have audit/read")
	}
}

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

// 通用 RBAC: 通配内置角色全权 + 作用域角色只在其策略内放行,无代码短路、无 governance 锁。
func TestRBAC_WildcardAdminAndScopedRole(t *testing.T) {
	a, cleanup := setupApp(t)
	defer cleanup()
	ctx := context.Background()

	// super_admin holds a '*'/'*' policy (seeded by migration 000010) → all-access,
	// including the formerly-locked audit/purge point. No code short-circuit involved.
	su, _ := mustCreateUser(t, a, "wildsuper")
	if err := a.IAM.GrantPlatformRoleByCode(ctx, kernel.ID(su), "super_admin", ""); err != nil {
		t.Fatalf("grant super_admin: %v", err)
	}
	for _, ra := range [][2]string{{"org", "create"}, {"authz", "grant"}, {"audit", "read"}, {"audit", "purge"}, {"anything", "whatever"}} {
		if err := a.IAM.EnforcePlatform(ctx, kernel.ID(su), ra[0], ra[1]); err != nil {
			t.Fatalf("super_admin (wildcard) must allow %s/%s: %v", ra[0], ra[1], err)
		}
	}

	// A scoped role (sys_admin) is allowed only within its seeded policy set.
	sys, _ := mustCreateUser(t, a, "scoped")
	if err := a.IAM.GrantPlatformRoleByCode(ctx, kernel.ID(sys), "sys_admin", ""); err != nil {
		t.Fatalf("grant sys_admin: %v", err)
	}
	if err := a.IAM.EnforcePlatform(ctx, kernel.ID(sys), "org", "create"); err != nil {
		t.Fatalf("sys_admin should allow org/create: %v", err)
	}
	if err := a.IAM.EnforcePlatform(ctx, kernel.ID(sys), "authz", "grant"); err == nil {
		t.Fatal("sys_admin must NOT have authz/grant (outside its scope)")
	}
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

// 命门 4: 回填 — 现有 is_platform_admin 用户(seeded admin)持有 super_admin
func TestPlatformKeystone_BackfilledAdminIsSuper(t *testing.T) {
	a, cleanup := setupApp(t)
	defer cleanup()
	ctx := context.Background()

	// Resolve the seeded bootstrap admin's id directly from the DB (robust, no
	// dependency on a specific iam lookup method name).
	var adminID string
	if err := a.Pool.QueryRow(ctx, `SELECT id::text FROM public.platform_user WHERE username = 'admin'`).Scan(&adminID); err != nil {
		t.Fatalf("seeded admin not found: %v", err)
	}
	if !a.IAM.UserHasPlatformRole(ctx, kernel.ID(adminID), "super_admin") {
		t.Fatal("backfilled admin must hold super_admin")
	}
}
