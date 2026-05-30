package iam

import (
	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
)

// RegisterAdminRoutes mounts /admin/roles + /me/* personal endpoints.
// Roles routes are admin-gated by caller. /me/* require only JWTAuth.
func RegisterAdminRoutes(r *gin.RouterGroup, svc *Service) {
	// Roles (admin)
	r.GET("/admin/roles", func(c *gin.Context) {
		tid, _ := kernel.TenantIDFromContext(c.Request.Context())
		roles, err := svc.ListRoles(c.Request.Context(), tid)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"roles": roles})
	})

	r.POST("/admin/roles", func(c *gin.Context) {
		tid, _ := kernel.TenantIDFromContext(c.Request.Context())
		var req struct{ Code, Name string }
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_request", "请求格式错误", err))
			return
		}
		role, err := svc.CreateRole(c.Request.Context(), CreateRoleCmd{
			TenantID: tid, Code: req.Code, Name: req.Name,
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.Created(c, role)
	})

	r.DELETE("/admin/roles/:id", func(c *gin.Context) {
		tid, _ := kernel.TenantIDFromContext(c.Request.Context())
		rid, _ := kernel.ParseID(c.Param("id"))
		if err := svc.DeleteRole(c.Request.Context(), tid, rid); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.POST("/admin/roles/:id/policies", func(c *gin.Context) {
		rid, _ := kernel.ParseID(c.Param("id"))
		var req struct{ Resource, Action string }
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_request", "请求格式错误", err))
			return
		}
		if err := svc.AddPolicy(c.Request.Context(), rid, req.Resource, req.Action); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.DELETE("/admin/roles/:id/policies", func(c *gin.Context) {
		rid, _ := kernel.ParseID(c.Param("id"))
		resource := c.Query("resource")
		action := c.Query("action")
		if err := svc.RemovePolicy(c.Request.Context(), rid, resource, action); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.POST("/admin/members/:id/roles", func(c *gin.Context) {
		tid, _ := kernel.TenantIDFromContext(c.Request.Context())
		mid, _ := kernel.ParseID(c.Param("id"))
		var req struct{ Code string }
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_request", "请求格式错误", err))
			return
		}
		if err := svc.GrantRoleByCode(c.Request.Context(), mid, tid, req.Code); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.DELETE("/admin/members/:id/roles/:roleId", func(c *gin.Context) {
		tid, _ := kernel.TenantIDFromContext(c.Request.Context())
		mid, _ := kernel.ParseID(c.Param("id"))
		rid, _ := kernel.ParseID(c.Param("roleId"))
		if err := svc.RevokeRole(c.Request.Context(), mid, tid, rid); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.GET("/admin/members/:id/roles", func(c *gin.Context) {
		tid, _ := kernel.TenantIDFromContext(c.Request.Context())
		mid, _ := kernel.ParseID(c.Param("id"))
		roles, err := svc.MemberRoles(c.Request.Context(), mid, tid)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"roles": roles})
	})
}

// RegisterMeRoutes mounts personal endpoints under /me/*. Requires only JWTAuth.
func RegisterMeRoutes(r *gin.RouterGroup, svc *Service) {
	r.POST("/me/password", func(c *gin.Context) {
		claims, _ := ClaimsFromContext(c.Request.Context())
		var req struct{ Old, New string }
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_request", "请求格式错误", err))
			return
		}
		if err := svc.ChangePassword(c.Request.Context(), ChangePasswordCmd{
			PlatformUserID: claims.PlatformUserID, Old: req.Old, New: req.New,
		}); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.GET("/me/sessions", func(c *gin.Context) {
		claims, _ := ClaimsFromContext(c.Request.Context())
		sessions, err := svc.ListSessions(c.Request.Context(), claims.PlatformUserID, claims.SessionID)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"sessions": sessions})
	})

	r.POST("/me/sessions/:id/revoke", func(c *gin.Context) {
		sid, _ := kernel.ParseID(c.Param("id"))
		if err := svc.Logout(c.Request.Context(), sid); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.GET("/me/admin", func(c *gin.Context) {
		claims, _ := ClaimsFromContext(c.Request.Context())
		tid := claims.TenantID
		isTA := svc.IsTenantAdmin(c.Request.Context(), claims.MemberID, tid)
		isPA := svc.IsPlatformAdmin(c.Request.Context(), claims.MemberID, tid)
		apiresp.OK(c, gin.H{
			"is_tenant_admin":   isTA,
			"is_platform_admin": isPA,
		})
	})
}

// TenantAdminRequired middleware ensures member has tenant_admin or platform_admin.
func TenantAdminRequired(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := ClaimsFromContext(c.Request.Context())
		if !ok || claims.MemberID == "" || claims.TenantID == "" {
			apiresp.Fail(c, errors.New(errors.KindForbidden, "iam.admin_required", "请使用管理员账号"))
			return
		}
		if !svc.IsTenantAdmin(c.Request.Context(), claims.MemberID, claims.TenantID) {
			apiresp.Fail(c, errors.New(errors.KindForbidden, "iam.admin_required", "需要租户管理员权限"))
			return
		}
		c.Next()
	}
}

// PlatformAdminRequired middleware ensures member has platform_admin role.
func PlatformAdminRequired(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := ClaimsFromContext(c.Request.Context())
		if !ok || claims.MemberID == "" || claims.TenantID == "" {
			apiresp.Fail(c, errors.New(errors.KindForbidden, "iam.platform_admin_required", "请使用平台管理员账号"))
			return
		}
		if !svc.IsPlatformAdmin(c.Request.Context(), claims.MemberID, claims.TenantID) {
			apiresp.Fail(c, errors.New(errors.KindForbidden, "iam.platform_admin_required", "需要平台管理员权限"))
			return
		}
		c.Next()
	}
}
