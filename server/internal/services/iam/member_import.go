package iam

import (
	"context"
	"crypto/rand"
	stderrors "errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/services/tenancy"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/tenantdb"
)

// registerMemberImportExportRoutes mounts member CSV export/template/import under
// /admin (the TenantAdminRequired group), each additionally gated with member:write.
func registerMemberImportExportRoutes(r *gin.RouterGroup, svc *Service, pool *pgxpool.Pool, authz AuthzFunc) {
	tenantOf := func(c *gin.Context) (*tenancy.Tenant, bool) {
		tc, ok := tenantdb.FromContext(c.Request.Context())
		if !ok || tc.ID == "" {
			return nil, false
		}
		return &tenancy.Tenant{ID: kernel.ID(tc.ID), Slug: tc.Slug, SchemaName: tc.SchemaName, Status: tc.Status}, true
	}

	r.GET("/admin/members/export", authz("member", "write"), func(c *gin.Context) {
		t, ok := tenantOf(c)
		if !ok {
			apiresp.Fail(c, errors.New(errors.KindForbidden, "iam.no_tenant", "缺少租户上下文"))
			return
		}
		rows, err := svc.tenants.ExportMembers(c.Request.Context(), pool, t)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.CSV(c, "members.csv", rows)
	})

	r.GET("/admin/members/template", authz("member", "write"), func(c *gin.Context) {
		apiresp.CSV(c, "members_template.csv", tenancy.MemberTemplateRows())
	})

	r.POST("/admin/members/import", authz("member", "write"), func(c *gin.Context) {
		t, ok := tenantOf(c)
		if !ok {
			apiresp.Fail(c, errors.New(errors.KindForbidden, "iam.no_tenant", "缺少租户上下文"))
			return
		}
		records, err := apiresp.ParseCSVUpload(c, "file")
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		res, err := svc.ImportMembers(c.Request.Context(), pool, t, records)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, res)
	})
}

// generateInitialPassword returns a random password that satisfies the strength
// rules (>=12 chars, mixed case, digit, symbol). Imported users are created with
// password_must_change=true, so this value is never shown — it only has to be a
// valid placeholder they must rotate on first login.
func generateInitialPassword() string {
	const (
		lower   = "abcdefghijkmnpqrstuvwxyz"
		upper   = "ABCDEFGHJKLMNPQRSTUVWXYZ"
		digits  = "23456789"
		symbols = "!@#$%^&*"
		all     = lower + upper + digits + symbols
	)
	pick := func(set string) byte {
		b := make([]byte, 1)
		_, _ = rand.Read(b)
		return set[int(b[0])%len(set)]
	}
	out := []byte{pick(upper), pick(lower), pick(digits), pick(symbols)}
	for len(out) < 16 {
		out = append(out, pick(all))
	}
	return string(out)
}

// ImportMembers bulk-creates/updates members of a tenant from parsed CSV records
// (header first). For each row it finds-or-creates the platform_user by username
// (new users get a forced-change initial password + username/phone, email is
// optional), joins them to the tenant with display_name/phone/title, resolves the
// department column to a dept_id, and grants the default tenant_member role.
//
// Idempotent by username: an already-existing membership has its display_name /
// phone / title / department updated rather than duplicated. Validation failures
// (missing/invalid username, unknown department) are recorded per-row; the import
// never 500s on bad data — partial success is reported in the BulkResult.
func (s *Service) ImportMembers(ctx context.Context, pool *pgxpool.Pool, t *tenancy.Tenant, records [][]string) (*kernel.BulkResult, error) {
	res := kernel.NewBulkResult()
	if len(records) == 0 {
		return res, nil
	}
	dataRecords := records
	if strings.EqualFold(strings.TrimSpace(memberCol(records[0], 0)), "username") {
		dataRecords = records[1:]
	}

	deptByName, err := s.tenants.DeptNameToID(ctx, pool, t)
	if err != nil {
		return nil, err
	}

	for i, rec := range dataRecords {
		lineNo := i + 1
		res.Total++
		username := strings.TrimSpace(memberCol(rec, 0))
		if username == "" {
			res.Fail(lineNo, "", "用户名不能为空")
			continue
		}
		if !usernameRe.MatchString(username) {
			res.Fail(lineNo, username, "用户名格式非法（3-32 位，小写字母/数字/-/_，字母开头）")
			continue
		}
		displayName := strings.TrimSpace(memberCol(rec, 1))
		if displayName == "" {
			displayName = username
		}
		phone := strings.TrimSpace(memberCol(rec, 2))
		email := strings.TrimSpace(memberCol(rec, 3))
		deptName := strings.TrimSpace(memberCol(rec, 4))
		title := strings.TrimSpace(memberCol(rec, 5))

		var deptID *kernel.ID
		if deptName != "" {
			did, ok := deptByName[deptName]
			if !ok {
				res.Fail(lineNo, username, fmt.Sprintf("部门 %q 不存在", deptName))
				continue
			}
			deptID = &did
		}

		// Find-or-create the platform_user by username.
		u, err := s.repo.GetUserByUsername(ctx, username)
		if err != nil {
			res.Fail(lineNo, username, "查询用户失败: "+err.Error())
			continue
		}
		if u == nil {
			u, err = s.RegisterUser(ctx, RegisterCmd{
				Username: username, Phone: phone, Email: email,
				Password: generateInitialPassword(),
			})
			if err != nil {
				res.Fail(lineNo, username, friendlyImportErr(err))
				continue
			}
			if err := s.repo.SetPasswordMustChange(ctx, u.ID, true); err != nil {
				_ = s.bus.Publish(ctx, "iam.set_must_change_failed", map[string]any{"platform_user_id": u.ID})
			}
		}

		// Join (idempotent — returns the existing membership if already a member).
		mem, err := s.tenants.JoinMember(ctx, pool, tenancy.JoinMemberCmd{
			PlatformUserID: u.ID, TenantID: t.ID,
			DisplayName: displayName, Phone: phone, Title: title,
		})
		if err != nil {
			res.Fail(lineNo, username, "加入组织失败: "+err.Error())
			continue
		}

		// JoinMember does an INSERT ... ON CONFLICT DO NOTHING, so for an existing
		// member the display_name/phone/title/dept must be applied via an update.
		if err := s.tenants.UpdateMember(ctx, pool, t, tenancy.UpdateMemberCmd{
			MemberID:    mem.MemberID,
			DisplayName: &displayName,
			Phone:       &phone,
			Title:       &title,
			DeptID:      deptID,
			SetDept:     true,
		}); err != nil {
			res.Fail(lineNo, username, "更新成员资料失败: "+err.Error())
			continue
		}

		if err := s.GrantRoleByCode(ctx, mem.MemberID, t.ID, "tenant_member"); err != nil {
			res.Fail(lineNo, username, "授予默认角色失败: "+err.Error())
			continue
		}
		res.Ok()
	}
	return res, nil
}

// friendlyImportErr surfaces the user-facing message from a known iam error,
// falling back to the raw error string.
func friendlyImportErr(err error) string {
	var e *errors.Error
	if stderrors.As(err, &e) && e.Message != "" {
		return e.Message
	}
	return err.Error()
}

func memberCol(rec []string, i int) string {
	if i < len(rec) {
		return rec[i]
	}
	return ""
}
