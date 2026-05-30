package tenancy

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/tenantdb"
)

// RegisterAdminRoutes mounts /admin/members, /admin/tenant under an auth+tenant group.
// Caller is responsible for gating these with tenant_admin RBAC.
func RegisterAdminRoutes(r *gin.RouterGroup, svc *Service, pool *pgxpool.Pool) {
	r.GET("/admin/members", func(c *gin.Context) {
		tc, _ := tenantdb.FromContext(c.Request.Context())
		var p kernel.Pagination
		_ = c.ShouldBindQuery(&p)
		t := &Tenant{ID: kernel.ID(tc.ID), Slug: tc.Slug, SchemaName: tc.SchemaName, Status: tc.Status}
		members, err := svc.ListMembers(c.Request.Context(), pool, t, p, c.Query("search"))
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"members": members})
	})

	r.PATCH("/admin/members/:id", func(c *gin.Context) {
		tc, _ := tenantdb.FromContext(c.Request.Context())
		mid, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		var req struct {
			DisplayName *string `json:"display_name"`
			Department  *string `json:"department"`
			Title       *string `json:"title"`
			Phone       *string `json:"phone"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_request", "请求格式错误", err))
			return
		}
		t := &Tenant{ID: kernel.ID(tc.ID), Slug: tc.Slug, SchemaName: tc.SchemaName, Status: tc.Status}
		if err := svc.UpdateMember(c.Request.Context(), pool, t, UpdateMemberCmd{
			MemberID: mid, DisplayName: req.DisplayName,
			Department: req.Department, Title: req.Title, Phone: req.Phone,
		}); err != nil {
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
}
