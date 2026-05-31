package iam

import (
	"testing"

	"github.com/leo/iop/server/internal/shared/kernel"
)

func TestMemberImportParserReportsExcelRowNumber(t *testing.T) {
	res := kernel.NewBulkResult()
	rows := parseMemberImportRows([][]string{
		{"username", "display_name", "phone", "email", "gender", "org_code", "post_code", "role_code", "status", "initial_password", "remark"},
		{"zhangsan", "张三", "13800000001", "zhangsan@example.com", "male", "RD", "engineer", "tenant_member", "active", "", ""},
		{"bad user", "坏账号", "", "", "", "RD", "", "", "active", "", ""},
	}, res)
	if len(rows) != 1 {
		t.Fatalf("expected 1 valid row, got %d", len(rows))
	}
	if res.Failed != 1 || len(res.Errors) != 1 {
		t.Fatalf("expected one validation error, got failed=%d errors=%v", res.Failed, res.Errors)
	}
	if res.Errors[0].Row != 3 {
		t.Fatalf("expected Excel row 3, got %d", res.Errors[0].Row)
	}
}

func TestMemberImportRejectsDuplicateUsernames(t *testing.T) {
	res := kernel.NewBulkResult()
	rows := parseMemberImportRows([][]string{
		{"username", "display_name", "phone", "email", "gender", "org_code", "post_code", "role_code", "status", "initial_password", "remark"},
		{"zhangsan", "张三", "13800000001", "", "", "RD", "", "", "active", "", ""},
		{"zhangsan", "张三二", "13800000002", "", "", "RD", "", "", "active", "", ""},
	}, res)
	failDuplicateImportUsernames(rows, res)
	if res.Failed != 1 {
		t.Fatalf("expected duplicate failure, got %d", res.Failed)
	}
	if res.Errors[0].Row != 3 {
		t.Fatalf("expected duplicate row 3, got %d", res.Errors[0].Row)
	}
}

func TestTenantMemberRoleGrantAllowedBlocksPlatformRoles(t *testing.T) {
	for _, code := range []string{"super_admin", "platform_admin", "sys_admin", "sec_admin", "audit_admin"} {
		if tenantMemberRoleGrantAllowed(code) {
			t.Fatalf("expected %s to be blocked", code)
		}
	}
	if !tenantMemberRoleGrantAllowed("tenant_member") {
		t.Fatal("tenant_member should be grantable")
	}
}
