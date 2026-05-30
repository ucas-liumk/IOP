package integration

import (
	"context"
	"testing"

	"github.com/leo/iop/server/internal/services/iam"
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
	_ = iam.PlatformPermission{}
}
