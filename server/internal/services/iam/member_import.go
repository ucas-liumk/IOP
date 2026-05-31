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

// registerMemberImportExportRoutes mounts member create/export/template/import under
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
		cmd, ok := memberListCmdFromQuery(c)
		if !ok {
			return
		}
		rows, err := svc.tenants.ExportMembers(c.Request.Context(), pool, t, cmd)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.XLSX(c, "members.xlsx", rows)
	})

	r.GET("/admin/members/template", authz("member", "write"), func(c *gin.Context) {
		apiresp.XLSX(c, "members_template.xlsx", tenancy.MemberTemplateRows())
	})

	r.POST("/admin/members/import", authz("member", "write"), func(c *gin.Context) {
		t, ok := tenantOf(c)
		if !ok {
			apiresp.Fail(c, errors.New(errors.KindForbidden, "iam.no_tenant", "缺少租户上下文"))
			return
		}
		records, err := apiresp.ParseTabularUpload(c, "file")
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		res, err := svc.ImportMembers(c.Request.Context(), pool, t, records, memberImportOptionsFromRequest(c))
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, res)
	})

	r.POST("/admin/members", authz("member", "write"), func(c *gin.Context) {
		t, ok := tenantOf(c)
		if !ok {
			apiresp.Fail(c, errors.New(errors.KindForbidden, "iam.no_tenant", "缺少租户上下文"))
			return
		}
		var req memberCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_request", "请求格式错误", err))
			return
		}
		cmd, ok := createMemberCmdFromRequest(c, req)
		if !ok {
			return
		}
		row, err := svc.CreateTenantMember(c.Request.Context(), pool, t, cmd)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, row)
	})

	r.POST("/admin/members/:id/reset-password", authz("member", "write"), func(c *gin.Context) {
		t, ok := tenantOf(c)
		if !ok {
			apiresp.Fail(c, errors.New(errors.KindForbidden, "iam.no_tenant", "缺少租户上下文"))
			return
		}
		mid, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_id", "成员 ID 无效", err))
			return
		}
		var req struct {
			NewPassword string `json:"new_password"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_request", "请求格式错误", err))
			return
		}
		row, err := svc.tenants.GetMember(c.Request.Context(), pool, t, mid)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		if svc.IsPlatformAdminUser(c.Request.Context(), row.PlatformUserID) {
			apiresp.Fail(c, errors.New(errors.KindForbidden, "iam.platform_admin_locked", "不能操作平台管理员账号"))
			return
		}
		if err := svc.AdminResetPassword(c.Request.Context(), row.PlatformUserID, req.NewPassword); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
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

type MemberImportOptions struct {
	DryRun bool
	Mode   string // overwrite (default) or skip
}

type memberCreateRequest struct {
	Username    string   `json:"username" binding:"required"`
	DisplayName string   `json:"display_name" binding:"required"`
	Phone       string   `json:"phone"`
	Email       string   `json:"email"`
	Gender      string   `json:"gender"`
	Title       string   `json:"title"`
	Remark      string   `json:"remark"`
	Password    string   `json:"password"`
	Status      string   `json:"status"`
	DeptID      string   `json:"dept_id" binding:"required"`
	RoleCodes   []string `json:"role_codes"`
	PostIDs     []string `json:"post_ids"`
}

func createMemberCmdFromRequest(c *gin.Context, req memberCreateRequest) (CreateTenantMemberCmd, bool) {
	deptID, err := kernel.ParseID(req.DeptID)
	if err != nil {
		apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_dept_id", "dept_id 无效", err))
		return CreateTenantMemberCmd{}, false
	}
	postIDs := []kernel.ID{}
	for _, raw := range req.PostIDs {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		id, err := kernel.ParseID(raw)
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_post_id", "post_id 无效", err))
			return CreateTenantMemberCmd{}, false
		}
		postIDs = append(postIDs, id)
	}
	return CreateTenantMemberCmd{
		Username:    req.Username,
		DisplayName: req.DisplayName,
		Phone:       req.Phone,
		Email:       req.Email,
		Gender:      req.Gender,
		Title:       req.Title,
		Remark:      req.Remark,
		Password:    req.Password,
		Status:      req.Status,
		DeptID:      deptID,
		RoleCodes:   req.RoleCodes,
		PostIDs:     postIDs,
	}, true
}

func memberListCmdFromQuery(c *gin.Context) (tenancy.ListMembersCmd, bool) {
	cmd := tenancy.ListMembersCmd{
		Search:  c.Query("search"),
		Status:  c.Query("status"),
		Subtree: c.Query("subtree") == "true",
	}
	if dq := c.Query("dept_id"); dq != "" {
		did, err := kernel.ParseID(dq)
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_dept_id", "dept_id 无效", err))
			return cmd, false
		}
		cmd.DeptID = &did
	}
	for _, raw := range splitCodes(c.Query("ids")) {
		id, err := kernel.ParseID(raw)
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_id", "用户 ID 无效", err))
			return cmd, false
		}
		cmd.IDs = append(cmd.IDs, id)
	}
	return cmd, true
}

func memberImportOptionsFromRequest(c *gin.Context) MemberImportOptions {
	return MemberImportOptions{
		DryRun: truthy(c.Query("dry_run")) || truthy(c.PostForm("dry_run")),
		Mode:   normalizeMemberImportMode(firstNonEmpty(c.Query("mode"), c.PostForm("mode"))),
	}
}

func normalizeMemberImportMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "skip":
		return "skip"
	default:
		return "overwrite"
	}
}

func truthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "y":
		return true
	default:
		return false
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// ImportMembers bulk-creates/updates members of a tenant from parsed Excel/CSV
// records. Every lookup is scoped to t: org_code resolves only in the target
// tenant, role_code resolves only among grantable tenant roles, and tenant admins
// cannot smuggle platform roles through the sheet.
func (s *Service) ImportMembers(ctx context.Context, pool *pgxpool.Pool, t *tenancy.Tenant, records [][]string, opts MemberImportOptions) (*kernel.BulkResult, error) {
	res := kernel.NewBulkResult()
	rows := parseMemberImportRows(records, res)
	if len(rows) == 0 {
		return res, nil
	}
	failDuplicateImportUsernames(rows, res)
	if res.Failed > 0 {
		return res, nil
	}
	mode := normalizeMemberImportMode(opts.Mode)
	deptByCode, err := s.tenants.DeptCodeToID(ctx, pool, t)
	if err != nil {
		return nil, err
	}
	postByCode, err := s.tenants.PostCodeToID(ctx, pool, t)
	if err != nil {
		return nil, err
	}
	roleByCode, err := s.grantableRoleCodes(ctx, t.ID)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		deptID, ok := deptByCode[strings.ToLower(row.orgCode)]
		if !ok {
			res.Fail(row.row, row.username, fmt.Sprintf("组织编码 %q 不存在或已停用", row.orgCode))
			continue
		}
		postIDs := []kernel.ID{}
		postOK := true
		for _, code := range splitCodes(row.postCode) {
			pid, ok := postByCode[strings.ToLower(code)]
			if !ok {
				res.Fail(row.row, row.username, fmt.Sprintf("岗位编码 %q 不存在或已停用", code))
				postOK = false
				break
			}
			postIDs = append(postIDs, pid)
		}
		if !postOK {
			continue
		}
		roleCodes := splitCodes(row.roleCode)
		if len(roleCodes) == 0 {
			roleCodes = []string{"tenant_member"}
		}
		roleOK := true
		for _, code := range roleCodes {
			if !tenantMemberRoleGrantAllowed(code) {
				res.Fail(row.row, row.username, "不能导入或分配平台管理员角色")
				roleOK = false
				break
			}
			if !roleByCode[strings.ToLower(code)] {
				res.Fail(row.row, row.username, fmt.Sprintf("角色编码 %q 不存在或不可分配", code))
				roleOK = false
				break
			}
		}
		if !roleOK {
			continue
		}
		u, err := s.repo.GetUserByUsername(ctx, row.username)
		if err != nil {
			res.Fail(row.row, row.username, "查询用户失败: "+err.Error())
			continue
		}
		var mem *tenancy.TenantMembership
		if u != nil {
			mem, err = s.tenants.GetMembership(ctx, u.ID, t.ID)
			if err != nil {
				res.Fail(row.row, row.username, "查询成员关系失败: "+err.Error())
				continue
			}
		}
		if u != nil && mode == "skip" {
			res.Ok()
			continue
		}
		if opts.DryRun {
			res.Ok()
			continue
		}
		if mem == nil {
			member, err := s.CreateTenantMember(ctx, pool, t, CreateTenantMemberCmd{
				Username:    row.username,
				DisplayName: row.displayName,
				Phone:       row.phone,
				Email:       row.email,
				Gender:      row.gender,
				Remark:      row.remark,
				Password:    row.initialPassword,
				Status:      row.status,
				DeptID:      deptID,
				RoleCodes:   roleCodes,
				PostIDs:     postIDs,
			})
			if err != nil {
				res.Fail(row.row, row.username, friendlyImportErr(err))
				continue
			}
			_ = member
			res.Ok()
			continue
		}
		deptIDCopy := deptID
		if err := s.tenants.UpdateMember(ctx, pool, t, tenancy.UpdateMemberCmd{
			MemberID:    mem.MemberID,
			DisplayName: &row.displayName,
			Phone:       &row.phone,
			Email:       &row.email,
			Gender:      &row.gender,
			Remark:      &row.remark,
			DeptID:      &deptIDCopy,
			SetDept:     true,
		}); err != nil {
			res.Fail(row.row, row.username, "更新成员资料失败: "+friendlyImportErr(err))
			continue
		}
		for _, code := range roleCodes {
			if err := s.GrantRoleByCode(ctx, mem.MemberID, t.ID, code); err != nil {
				res.Fail(row.row, row.username, "授予角色失败: "+friendlyImportErr(err))
				roleOK = false
				break
			}
		}
		if !roleOK {
			continue
		}
		postOK = true
		for _, postID := range postIDs {
			if err := s.tenants.AssignMemberPost(ctx, pool, t, mem.MemberID, postID); err != nil {
				res.Fail(row.row, row.username, "分配岗位失败: "+friendlyImportErr(err))
				postOK = false
				break
			}
		}
		if !postOK {
			continue
		}
		if row.status != "" {
			if err := s.tenants.SetMemberStatus(ctx, pool, t, mem.MemberID, row.status); err != nil {
				res.Fail(row.row, row.username, "更新状态失败: "+friendlyImportErr(err))
				continue
			}
		}
		if row.initialPassword != "" && u != nil {
			if err := s.AdminResetPassword(ctx, u.ID, row.initialPassword); err != nil {
				res.Fail(row.row, row.username, "重置密码失败: "+friendlyImportErr(err))
				continue
			}
		}
		res.Ok()
	}
	return res, nil
}

type memberImportRow struct {
	row             int
	username        string
	displayName     string
	phone           string
	email           string
	gender          string
	orgCode         string
	postCode        string
	roleCode        string
	status          string
	initialPassword string
	remark          string
}

func parseMemberImportRows(records [][]string, res *kernel.BulkResult) []memberImportRow {
	if len(records) == 0 {
		return nil
	}
	start := 0
	if strings.EqualFold(strings.TrimSpace(memberCol(records[0], 0)), "username") {
		start = 1
	}
	rows := []memberImportRow{}
	for i := start; i < len(records); i++ {
		rec := records[i]
		if rowBlank(rec) {
			continue
		}
		lineNo := i + 1
		res.Total++
		row := memberImportRow{
			row:             lineNo,
			username:        strings.TrimSpace(memberCol(rec, 0)),
			displayName:     strings.TrimSpace(memberCol(rec, 1)),
			phone:           strings.TrimSpace(memberCol(rec, 2)),
			email:           strings.ToLower(strings.TrimSpace(memberCol(rec, 3))),
			gender:          strings.TrimSpace(memberCol(rec, 4)),
			orgCode:         strings.TrimSpace(memberCol(rec, 5)),
			postCode:        strings.TrimSpace(memberCol(rec, 6)),
			roleCode:        strings.TrimSpace(memberCol(rec, 7)),
			status:          normalizeMemberStatus(memberCol(rec, 8)),
			initialPassword: strings.TrimSpace(memberCol(rec, 9)),
			remark:          strings.TrimSpace(memberCol(rec, 10)),
		}
		switch {
		case row.username == "":
			res.Fail(lineNo, "", "用户名不能为空")
			continue
		case !usernameRe.MatchString(row.username):
			res.Fail(lineNo, row.username, "用户名格式非法（3-32 位，小写字母/数字/-/_，字母开头）")
			continue
		case row.displayName == "":
			res.Fail(lineNo, row.username, "姓名不能为空")
			continue
		case row.orgCode == "":
			res.Fail(lineNo, row.username, "组织编码不能为空")
			continue
		case row.phone != "" && !phoneRe.MatchString(row.phone):
			res.Fail(lineNo, row.username, "手机号格式错误")
			continue
		case row.email != "" && !strings.Contains(row.email, "@"):
			res.Fail(lineNo, row.username, "邮箱格式错误")
			continue
		case row.status != "active" && row.status != "disabled":
			res.Fail(lineNo, row.username, "状态只能是 active 或 disabled")
			continue
		}
		rows = append(rows, row)
	}
	return rows
}

func failDuplicateImportUsernames(rows []memberImportRow, res *kernel.BulkResult) {
	first := map[string]int{}
	for _, row := range rows {
		key := strings.ToLower(row.username)
		if prev, ok := first[key]; ok {
			res.Fail(row.row, row.username, fmt.Sprintf("用户名与第 %d 行重复", prev))
			continue
		}
		first[key] = row.row
	}
}

func rowBlank(rec []string) bool {
	for _, c := range rec {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

func normalizeMemberStatus(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	switch v {
	case "", "normal", "enabled":
		return "active"
	case "disable", "disabled", "停用", "禁用":
		return "disabled"
	default:
		return v
	}
}

func splitCodes(v string) []string {
	v = strings.NewReplacer("，", ",", "；", ",", ";", ",", "\n", ",").Replace(v)
	parts := strings.Split(v, ",")
	out := []string{}
	seen := map[string]bool{}
	for _, p := range parts {
		code := strings.TrimSpace(p)
		if code == "" {
			continue
		}
		key := strings.ToLower(code)
		if !seen[key] {
			seen[key] = true
			out = append(out, code)
		}
	}
	return out
}

func (s *Service) grantableRoleCodes(ctx context.Context, tenantID kernel.ID) (map[string]bool, error) {
	roles, err := s.ListRoles(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, r := range roles {
		out[strings.ToLower(r.Code)] = true
	}
	return out, nil
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
