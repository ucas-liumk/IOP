package iam

import (
	"encoding/json"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/services/audit"
	"github.com/leo/iop/server/internal/services/tenancy"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
)

// orgTenantFromParam resolves the organization (tenant) named by the :tid path
// param. It writes a 4xx response and returns (nil,false) when the id is malformed
// or the org does not exist — callers should `return` on false.
//
// This is the bridge that lets the PLATFORM console (which carries NO tenant
// context of its own) operate on ANY organization's members: the caller passes the
// org's tenant id, we load the *Tenant (which holds the isolated SchemaName), and
// hand it straight to the existing member service funcs — they SET LOCAL
// search_path to that schema, so all reads/writes stay scoped to the target org.
// Identical in spirit to tenancy.orgTenantFromParam, but lives here because member
// management spans BOTH the tenancy service (members/posts) and the iam service
// (import / roles), and iam already imports tenancy (so wiring it here is acyclic;
// putting it in tenancy would require tenancy→iam, a cycle).
func orgTenantFromParam(c *gin.Context, svc *Service) (*tenancy.Tenant, bool) {
	id, err := kernel.ParseID(c.Param("tid"))
	if err != nil {
		apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_org_id", "组织 ID 无效", err))
		return nil, false
	}
	t, err := svc.tenants.GetTenant(c.Request.Context(), id)
	if err != nil {
		apiresp.Fail(c, err)
		return nil, false
	}
	if t == nil {
		apiresp.Fail(c, errors.New(errors.KindNotFound, "iam.org_not_found", "组织不存在"))
		return nil, false
	}
	return t, true
}

// RegisterPlatformOrgMemberRoutes mounts /platform/orgs/:tid/members/* (+ the
// per-org /posts and /roles pickers) on the platform group. The platform group has
// JWTAuth + PasswordChangeGate + PlatformAccess but NO TenantLoader, so each
// handler resolves the org from :tid and injects its schema via the resolved
// *Tenant — exactly mirroring tenancy.RegisterPlatformOrgRoutes (depts).
//
// Reads are gated PlatformAuthz user:read; mutations PlatformAuthz user:write.
// Every handler reuses the SAME service funcs the tenant console uses
// (svc.tenants.{ListMembers,ExportMembers,UpdateMember,AssignMemberPost,
// RemoveMemberPost,ListPosts} + svc.{ImportMembers,ListRoles,GrantRoleByCode,
// GrantRoleByID,RevokeRole}) — only the tenant resolution differs. No SQL is
// duplicated.
//
// Path note: members use the :mid param (and :pid/:rid for nested ids), distinct
// from the dept routes' :id, and live under the static /members/ segment, so they
// never collide with /depts/ in gin's routing tree.
func RegisterPlatformOrgMemberRoutes(r *gin.RouterGroup, svc *Service, aud *audit.Service, pool *pgxpool.Pool) {
	read := func() gin.HandlerFunc { return PlatformAuthz(svc, aud, "user", "read") }
	write := func() gin.HandlerFunc { return PlatformAuthz(svc, aud, "user", "write") }

	// --- Spreadsheet export / template / import. Registered before the parameterized
	// :mid routes so the literal sub-paths take precedence. ---

	r.GET("/platform/orgs/:tid/members/export", read(), func(c *gin.Context) {
		t, ok := orgTenantFromParam(c, svc)
		if !ok {
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

	r.GET("/platform/orgs/:tid/members/template", read(), func(c *gin.Context) {
		// Resolve the org so a bad/missing :tid still 404s consistently, even
		// though the template content itself is org-independent.
		if _, ok := orgTenantFromParam(c, svc); !ok {
			return
		}
		apiresp.XLSX(c, "members_template.xlsx", tenancy.MemberTemplateRows())
	})

	r.POST("/platform/orgs/:tid/members/import", write(), func(c *gin.Context) {
		t, ok := orgTenantFromParam(c, svc)
		if !ok {
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

	r.POST("/platform/orgs/:tid/members", write(), func(c *gin.Context) {
		t, ok := orgTenantFromParam(c, svc)
		if !ok {
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

	// --- Paged member listing. ---

	r.GET("/platform/orgs/:tid/members", read(), func(c *gin.Context) {
		t, ok := orgTenantFromParam(c, svc)
		if !ok {
			return
		}
		var p kernel.Pagination
		_ = c.ShouldBindQuery(&p)
		p = p.Normalize()
		if p.PageSize > 100 {
			p.PageSize = 100
		}
		cmd := tenancy.ListMembersCmd{
			Page:    p,
			Search:  c.Query("search"),
			Status:  c.Query("status"),
			Subtree: c.Query("subtree") == "true",
		}
		if dq := c.Query("dept_id"); dq != "" {
			did, err := kernel.ParseID(dq)
			if err != nil {
				apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_dept_id", "dept_id 无效", err))
				return
			}
			cmd.DeptID = &did
		}
		page, err := svc.tenants.ListMembers(c.Request.Context(), pool, t, cmd)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{
			"data":      page.Data,
			"members":   page.Data, // alias for parity with /admin/members
			"total":     page.Total,
			"page":      page.Page,
			"page_size": page.PageSize,
		})
	})

	r.GET("/platform/orgs/:tid/members/grouped", read(), func(c *gin.Context) {
		t, ok := orgTenantFromParam(c, svc)
		if !ok {
			return
		}
		cmd, ok := memberListCmdFromQuery(c)
		if !ok {
			return
		}
		tree, err := svc.tenants.GroupMembers(c.Request.Context(), pool, t, cmd)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"tree": tree})
	})

	r.GET("/platform/orgs/:tid/members/:mid", read(), func(c *gin.Context) {
		t, ok := orgTenantFromParam(c, svc)
		if !ok {
			return
		}
		mid, err := kernel.ParseID(c.Param("mid"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_id", "成员 ID 无效", err))
			return
		}
		row, err := svc.tenants.GetMember(c.Request.Context(), pool, t, mid)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, row)
	})

	// --- Per-member mutations. ---

	r.PATCH("/platform/orgs/:tid/members/:mid", write(), func(c *gin.Context) {
		t, ok := orgTenantFromParam(c, svc)
		if !ok {
			return
		}
		mid, err := kernel.ParseID(c.Param("mid"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_id", "成员 ID 无效", err))
			return
		}
		// Read the raw body so an absent dept_id (leave unchanged) is distinguishable
		// from an explicit null (clear) — a *kernel.ID alone cannot tell them apart.
		raw, _ := io.ReadAll(c.Request.Body)
		var req struct {
			DisplayName *string `json:"display_name"`
			Department  *string `json:"department"`
			Title       *string `json:"title"`
			Phone       *string `json:"phone"`
			Email       *string `json:"email"`
			Gender      *string `json:"gender"`
			Remark      *string `json:"remark"`
			Status      *string `json:"status"`
			DeptID      *string `json:"dept_id"`
		}
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &req); err != nil {
				apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_request", "请求格式错误", err))
				return
			}
		}
		cmd := tenancy.UpdateMemberCmd{
			MemberID: mid, DisplayName: req.DisplayName,
			Department: req.Department, Title: req.Title, Phone: req.Phone,
			Email: req.Email, Gender: req.Gender, Remark: req.Remark,
		}
		var present map[string]json.RawMessage
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &present)
		}
		if _, ok := present["dept_id"]; ok {
			cmd.SetDept = true
			if req.DeptID != nil && *req.DeptID != "" {
				did, err := kernel.ParseID(*req.DeptID)
				if err != nil {
					apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_dept_id", "dept_id 无效", err))
					return
				}
				cmd.DeptID = &did
			}
		}
		if err := svc.tenants.UpdateMember(c.Request.Context(), pool, t, cmd); err != nil {
			apiresp.Fail(c, err)
			return
		}
		if req.Status != nil {
			if err := svc.tenants.SetMemberStatus(c.Request.Context(), pool, t, mid, *req.Status); err != nil {
				apiresp.Fail(c, err)
				return
			}
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.POST("/platform/orgs/:tid/members/:mid/disable", write(), func(c *gin.Context) {
		t, ok := orgTenantFromParam(c, svc)
		if !ok {
			return
		}
		mid, err := kernel.ParseID(c.Param("mid"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_id", "成员 ID 无效", err))
			return
		}
		if err := svc.tenants.SetMemberStatus(c.Request.Context(), pool, t, mid, "disabled"); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.POST("/platform/orgs/:tid/members/:mid/enable", write(), func(c *gin.Context) {
		t, ok := orgTenantFromParam(c, svc)
		if !ok {
			return
		}
		mid, err := kernel.ParseID(c.Param("mid"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_id", "成员 ID 无效", err))
			return
		}
		if err := svc.tenants.SetMemberStatus(c.Request.Context(), pool, t, mid, "active"); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.POST("/platform/orgs/:tid/members/:mid/reset-password", write(), func(c *gin.Context) {
		t, ok := orgTenantFromParam(c, svc)
		if !ok {
			return
		}
		mid, err := kernel.ParseID(c.Param("mid"))
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
		if err := svc.AdminResetPassword(c.Request.Context(), row.PlatformUserID, req.NewPassword); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.POST("/platform/orgs/:tid/members/:mid/posts", write(), func(c *gin.Context) {
		t, ok := orgTenantFromParam(c, svc)
		if !ok {
			return
		}
		mid, err := kernel.ParseID(c.Param("mid"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_id", "成员 ID 无效", err))
			return
		}
		var req struct {
			PostID string `json:"post_id" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_request", "请求格式错误", err))
			return
		}
		pid, err := kernel.ParseID(req.PostID)
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_post_id", "post_id 无效", err))
			return
		}
		if err := svc.tenants.AssignMemberPost(c.Request.Context(), pool, t, mid, pid); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.DELETE("/platform/orgs/:tid/members/:mid/posts/:pid", write(), func(c *gin.Context) {
		t, ok := orgTenantFromParam(c, svc)
		if !ok {
			return
		}
		mid, err := kernel.ParseID(c.Param("mid"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_id", "成员 ID 无效", err))
			return
		}
		pid, err := kernel.ParseID(c.Param("pid"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_post_id", "post_id 无效", err))
			return
		}
		if err := svc.tenants.RemoveMemberPost(c.Request.Context(), pool, t, mid, pid); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	// Grant/revoke a role of THIS org to the member:
	// role_grant(role_id, member_id=:mid, tenant_id=:tid). Accepts either an
	// explicit role_id or a built-in role code.
	r.POST("/platform/orgs/:tid/members/:mid/roles", write(), func(c *gin.Context) {
		t, ok := orgTenantFromParam(c, svc)
		if !ok {
			return
		}
		mid, err := kernel.ParseID(c.Param("mid"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_id", "成员 ID 无效", err))
			return
		}
		var req struct {
			RoleID string `json:"role_id"`
			Code   string `json:"code"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_request", "请求格式错误", err))
			return
		}
		switch {
		case req.RoleID != "":
			rid, err := kernel.ParseID(req.RoleID)
			if err != nil {
				apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_role_id", "role_id 无效", err))
				return
			}
			if err := svc.GrantRoleByID(c.Request.Context(), mid, t.ID, rid); err != nil {
				apiresp.Fail(c, err)
				return
			}
		case req.Code != "":
			if err := svc.GrantRoleByCode(c.Request.Context(), mid, t.ID, req.Code); err != nil {
				apiresp.Fail(c, err)
				return
			}
		default:
			apiresp.Fail(c, errors.New(errors.KindParam, "iam.invalid_request", "role_id 或 code 必填"))
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.DELETE("/platform/orgs/:tid/members/:mid/roles/:rid", write(), func(c *gin.Context) {
		t, ok := orgTenantFromParam(c, svc)
		if !ok {
			return
		}
		mid, err := kernel.ParseID(c.Param("mid"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_id", "成员 ID 无效", err))
			return
		}
		rid, err := kernel.ParseID(c.Param("rid"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_role_id", "role_id 无效", err))
			return
		}
		if err := svc.RevokeRole(c.Request.Context(), mid, t.ID, rid); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	// Current roles granted to the member in THIS org — used to pre-check the
	// assign-role dialog. Mirrors /admin/members/:id/roles, but resolves the org
	// from :tid instead of tenant context.
	r.GET("/platform/orgs/:tid/members/:mid/roles", read(), func(c *gin.Context) {
		t, ok := orgTenantFromParam(c, svc)
		if !ok {
			return
		}
		mid, err := kernel.ParseID(c.Param("mid"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_id", "成员 ID 无效", err))
			return
		}
		roles, err := svc.MemberRoles(c.Request.Context(), mid, t.ID)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"roles": roles})
	})

	// --- Pickers for the assign-post / assign-role dialogs. ---

	r.GET("/platform/orgs/:tid/posts", read(), func(c *gin.Context) {
		t, ok := orgTenantFromParam(c, svc)
		if !ok {
			return
		}
		posts, err := svc.tenants.ListPosts(c.Request.Context(), pool, t)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"posts": posts})
	})

	r.GET("/platform/orgs/:tid/roles", read(), func(c *gin.Context) {
		t, ok := orgTenantFromParam(c, svc)
		if !ok {
			return
		}
		// ListRoles(tenantID) = platform-wide built-ins (tenant_id IS NULL) + this
		// org's own roles — exactly the set grantable to a member of this org.
		roles, err := svc.ListRoles(c.Request.Context(), t.ID)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"roles": roles})
	})
}
