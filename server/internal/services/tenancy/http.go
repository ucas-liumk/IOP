package tenancy

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
)

// RegisterPublicRoutes mounts unauthenticated lookup endpoints.
// Currently: GET /public/organizations — list of active tenants for the register dropdown.
// Only exposes id/name/slug (no schema_name, no internal state).
func RegisterPublicRoutes(r *gin.RouterGroup, svc *Service) {
	r.GET("/public/organizations", func(c *gin.Context) {
		ts, err := svc.ListActiveTenants(c.Request.Context(), kernel.Pagination{Page: 1, PageSize: 500})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		out := make([]gin.H, 0, len(ts))
		for _, t := range ts {
			out = append(out, gin.H{"id": t.ID, "name": t.Name, "slug": t.Slug})
		}
		apiresp.OK(c, gin.H{"organizations": out, "count": len(out)})
	})
}

// RegisterRoutes wires /tenants/* (platform admin).
// Caller is expected to gate these with RBAC platform_admin.
func RegisterRoutes(r *gin.RouterGroup, svc *Service, pool *pgxpool.Pool, authz AuthzFunc) {
	r.POST("/tenants", authz("org", "write"), func(c *gin.Context) {
		var req CreateTenantCmd
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_request", "请求格式错误", err))
			return
		}
		t, err := svc.CreateTenant(c.Request.Context(), req)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.Created(c, t)
	})

	r.GET("/tenants", authz("org", "read"), func(c *gin.Context) {
		var p kernel.Pagination
		_ = c.ShouldBindQuery(&p)
		ts, err := svc.ListActiveTenants(c.Request.Context(), p)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"tenants": ts})
	})

	r.GET("/tenants/:id", authz("org", "read"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_id", "id 格式错误", err))
			return
		}
		t, err := svc.GetTenant(c.Request.Context(), id)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		if t == nil {
			apiresp.Fail(c, errors.New(errors.KindNotFound, "tenancy.not_found", "租户不存在"))
			return
		}
		apiresp.OK(c, t)
	})

	r.POST("/tenants/:id/suspend", authz("org", "write"), func(c *gin.Context) {
		id, _ := kernel.ParseID(c.Param("id"))
		if err := svc.SuspendTenant(c.Request.Context(), id, c.Query("reason")); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.POST("/tenants/:id/resume", authz("org", "write"), func(c *gin.Context) {
		id, _ := kernel.ParseID(c.Param("id"))
		if err := svc.ResumeTenant(c.Request.Context(), id); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.POST("/tenants/:id/close", authz("org", "write"), func(c *gin.Context) {
		id, _ := kernel.ParseID(c.Param("id"))
		if err := svc.CloseTenant(c.Request.Context(), id); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.POST("/tenants/:id/members", authz("user", "write"), func(c *gin.Context) {
		id, _ := kernel.ParseID(c.Param("id"))
		var req struct {
			PlatformUserID string `json:"platform_user_id" binding:"required"`
			DisplayName    string `json:"display_name" binding:"required"`
			Email          string `json:"email" binding:"required"`
			Department     string `json:"department"`
			Title          string `json:"title"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_request", "请求格式错误", err))
			return
		}
		uid, err := kernel.ParseID(req.PlatformUserID)
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_user_id", "user id 格式错误", err))
			return
		}
		m, err := svc.JoinMember(c.Request.Context(), pool, JoinMemberCmd{
			PlatformUserID: uid, TenantID: id,
			DisplayName: req.DisplayName, Email: req.Email,
			Department: req.Department, Title: req.Title,
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.Created(c, m)
	})
}
