package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

	t.Cleanup(func() { _ = a.IAM.SetGovernanceMode(context.Background(), iam.ModeSingleAdmin, "") })

	_ = a.IAM.SetGovernanceMode(ctx, iam.ModeSingleAdmin, "")
	if err := a.IAM.EnforcePlatform(ctx, kernel.ID(su), "audit", "purge"); err != nil {
		t.Fatalf("single mode super should purge: %v", err)
	}
	_ = a.IAM.SetGovernanceMode(ctx, iam.ModeThreeMember, "")
	if err := a.IAM.EnforcePlatform(ctx, kernel.ID(su), "audit", "purge"); err == nil {
		t.Fatal("three_member: super_admin must NOT purge audit")
	}
	if err := a.IAM.EnforcePlatform(ctx, kernel.ID(su), "org", "create"); err != nil {
		t.Fatalf("super should still create orgs in three_member: %v", err)
	}
	_ = a.IAM.SetGovernanceMode(ctx, iam.ModeSingleAdmin, "") // restore (avoid leaking mode to other tests)
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

// 命门 6: 非超管无法切换治理模式 (HTTP level)
//
// Option A (real HTTP test): create a user → grant sys_admin (NOT super_admin) →
// login via POST /api/auth/login with email → use Bearer token to PUT
// /api/platform/rbac/governance-mode → assert 403 and that the mode is unchanged.
func TestPlatformKeystone_NonSuperCannotSwitchMode(t *testing.T) {
	a, cleanup := setupApp(t)
	defer cleanup()
	ctx := context.Background()

	// Spin up a real HTTP server backed by the full engine.
	srv := httptest.NewServer(a.Engine())
	defer srv.Close()

	// Restore governance mode to single_admin on test exit in case anything leaked.
	t.Cleanup(func() {
		_ = a.IAM.SetGovernanceMode(context.Background(), iam.ModeSingleAdmin, "")
	})

	// 1. Create a user and grant sys_admin (not super_admin).
	uid, email := mustCreateUser(t, a, "nonsupermode")
	if err := a.IAM.GrantPlatformRoleByCode(ctx, kernel.ID(uid), "sys_admin", ""); err != nil {
		t.Fatalf("grant sys_admin: %v", err)
	}

	// Sanity check: the user must NOT already hold super_admin.
	if a.IAM.UserHasPlatformRole(ctx, kernel.ID(uid), "super_admin") {
		t.Fatal("test setup error: sys_admin user must not have super_admin")
	}

	// 2. Login via HTTP to obtain a Bearer token.
	loginBody, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": "Test1234abc!",
	})
	loginResp, err := http.Post(srv.URL+"/api/auth/login", "application/json", bytes.NewReader(loginBody))
	if err != nil {
		t.Fatalf("login HTTP: %v", err)
	}
	defer loginResp.Body.Close()
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("login expected 200, got %d", loginResp.StatusCode)
	}
	var loginData struct {
		Data struct {
			Token struct {
				AccessToken string `json:"access_token"`
			} `json:"token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(loginResp.Body).Decode(&loginData); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	token := loginData.Data.Token.AccessToken
	if token == "" {
		t.Fatal("login returned empty access_token")
	}

	// 3. Ensure mode starts as single_admin.
	_ = a.IAM.SetGovernanceMode(ctx, iam.ModeSingleAdmin, "")

	// 4. Attempt PUT /api/platform/rbac/governance-mode with the sys_admin token.
	putBody, _ := json.Marshal(map[string]string{"mode": "three_member"})
	putReq, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/platform/rbac/governance-mode", bytes.NewReader(putBody))
	putReq.Header.Set("Content-Type", "application/json")
	putReq.Header.Set("Authorization", "Bearer "+token)
	putResp, err := http.DefaultClient.Do(putReq)
	if err != nil {
		t.Fatalf("PUT governance-mode HTTP: %v", err)
	}
	defer putResp.Body.Close()

	// 5. Assert the server returned 403.
	if putResp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for sys_admin switching mode, got %d", putResp.StatusCode)
	}

	// 6. Assert the mode was NOT changed (defence-in-depth: verify DB state).
	got := a.IAM.GovernanceMode(ctx)
	if got != iam.ModeSingleAdmin {
		t.Fatalf("mode must remain single_admin after rejected attempt, got %q", got)
	}
}
