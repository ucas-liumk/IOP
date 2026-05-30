package tenancy

import (
	"encoding/json"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/tenantdb"
)

// tenantFromCtx builds a *Tenant from the loaded TenantContext (mirrors the
// existing member handlers).
func tenantFromCtx(c *gin.Context) *Tenant {
	tc, _ := tenantdb.FromContext(c.Request.Context())
	return &Tenant{ID: kernel.ID(tc.ID), Slug: tc.Slug, SchemaName: tc.SchemaName, Status: tc.Status}
}

// AuthzFunc returns a per-route RBAC gate for (resource, action). Supplied by the
// caller (app wiring) so this package needn't import iam.
type AuthzFunc func(resource, action string) gin.HandlerFunc

// RegisterAdminRoutes mounts /admin/members, /admin/tenant under an auth+tenant group.
// Caller is responsible for gating these with tenant_admin RBAC; authz adds the
// per-route permission gate for import/export (dept:write / member:write etc).
func RegisterAdminRoutes(r *gin.RouterGroup, svc *Service, pool *pgxpool.Pool, authz AuthzFunc) {
	r.GET("/admin/members", func(c *gin.Context) {
		tc, _ := tenantdb.FromContext(c.Request.Context())
		var p kernel.Pagination
		_ = c.ShouldBindQuery(&p)
		// Per-endpoint ceiling of 100 (tighter than kernel's global clamp).
		p = p.Normalize()
		if p.PageSize > 100 {
			p.PageSize = 100
		}
		t := &Tenant{ID: kernel.ID(tc.ID), Slug: tc.Slug, SchemaName: tc.SchemaName, Status: tc.Status}
		cmd := ListMembersCmd{Page: p, Search: c.Query("search"), Subtree: c.Query("subtree") == "true"}
		if dq := c.Query("dept_id"); dq != "" {
			did, err := kernel.ParseID(dq)
			if err != nil {
				apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_dept_id", "dept_id 无效", err))
				return
			}
			cmd.DeptID = &did
		}
		page, err := svc.ListMembers(c.Request.Context(), pool, t, cmd)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		// Paginated envelope (data/total/page/page_size). "members" is kept as an
		// alias of data for back-compat with the existing frontend.
		apiresp.OK(c, gin.H{
			"members":   page.Data,
			"data":      page.Data,
			"total":     page.Total,
			"page":      page.Page,
			"page_size": page.PageSize,
		})
	})

	r.PATCH("/admin/members/:id", func(c *gin.Context) {
		tc, _ := tenantdb.FromContext(c.Request.Context())
		mid, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		// Read the raw body so we can distinguish an absent dept_id (leave unchanged)
		// from an explicit null (clear) — a *kernel.ID alone cannot tell them apart.
		raw, _ := io.ReadAll(c.Request.Body)
		var req struct {
			DisplayName *string `json:"display_name"`
			Department  *string `json:"department"`
			Title       *string `json:"title"`
			Phone       *string `json:"phone"`
			DeptID      *string `json:"dept_id"`
		}
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &req); err != nil {
				apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_request", "请求格式错误", err))
				return
			}
		}
		cmd := UpdateMemberCmd{
			MemberID: mid, DisplayName: req.DisplayName,
			Department: req.Department, Title: req.Title, Phone: req.Phone,
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
					apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_dept_id", "dept_id 无效", err))
					return
				}
				cmd.DeptID = &did
			}
		}
		t := &Tenant{ID: kernel.ID(tc.ID), Slug: tc.Slug, SchemaName: tc.SchemaName, Status: tc.Status}
		if err := svc.UpdateMember(c.Request.Context(), pool, t, cmd); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.POST("/admin/members/:id/posts", func(c *gin.Context) {
		tc, _ := tenantdb.FromContext(c.Request.Context())
		mid, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_id", "成员 ID 无效", err))
			return
		}
		var req struct {
			PostID string `json:"post_id" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_request", "请求格式错误", err))
			return
		}
		pid, err := kernel.ParseID(req.PostID)
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_post_id", "post_id 无效", err))
			return
		}
		t := &Tenant{ID: kernel.ID(tc.ID), Slug: tc.Slug, SchemaName: tc.SchemaName, Status: tc.Status}
		if err := svc.AssignMemberPost(c.Request.Context(), pool, t, mid, pid); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.DELETE("/admin/members/:id/posts/:postId", func(c *gin.Context) {
		tc, _ := tenantdb.FromContext(c.Request.Context())
		mid, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_id", "成员 ID 无效", err))
			return
		}
		pid, err := kernel.ParseID(c.Param("postId"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_post_id", "post_id 无效", err))
			return
		}
		t := &Tenant{ID: kernel.ID(tc.ID), Slug: tc.Slug, SchemaName: tc.SchemaName, Status: tc.Status}
		if err := svc.RemoveMemberPost(c.Request.Context(), pool, t, mid, pid); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.POST("/admin/members/:id/disable", func(c *gin.Context) {
		tc, _ := tenantdb.FromContext(c.Request.Context())
		mid, _ := kernel.ParseID(c.Param("id"))
		t := &Tenant{ID: kernel.ID(tc.ID), Slug: tc.Slug, SchemaName: tc.SchemaName, Status: tc.Status}
		if err := svc.SetMemberStatus(c.Request.Context(), pool, t, mid, "disabled"); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.POST("/admin/members/:id/enable", func(c *gin.Context) {
		tc, _ := tenantdb.FromContext(c.Request.Context())
		mid, _ := kernel.ParseID(c.Param("id"))
		t := &Tenant{ID: kernel.ID(tc.ID), Slug: tc.Slug, SchemaName: tc.SchemaName, Status: tc.Status}
		if err := svc.SetMemberStatus(c.Request.Context(), pool, t, mid, "active"); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.GET("/admin/tenant", func(c *gin.Context) {
		tc, _ := tenantdb.FromContext(c.Request.Context())
		t, err := svc.GetTenant(c.Request.Context(), kernel.ID(tc.ID))
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		count, _ := svc.CountMembers(c.Request.Context(), pool,
			&Tenant{ID: kernel.ID(tc.ID), Slug: tc.Slug, SchemaName: tc.SchemaName, Status: tc.Status})
		apiresp.OK(c, gin.H{"tenant": t, "member_count": count})
	})

	r.PATCH("/admin/tenant", func(c *gin.Context) {
		tc, _ := tenantdb.FromContext(c.Request.Context())
		var req struct {
			Name string `json:"name"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_request", "请求格式错误", err))
			return
		}
		if err := svc.UpdateName(c.Request.Context(), kernel.ID(tc.ID), req.Name); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	registerDeptRoutes(r, svc, pool, authz)
	registerPostRoutes(r, svc, pool)
	registerNoticeRoutes(r, svc, pool)
}
