package iam

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/tenantdb"
)

// parseIDs converts a slice of string ids into kernel.IDs, failing fast on a bad value.
func parseIDs(ss []string) ([]kernel.ID, error) {
	out := make([]kernel.ID, 0, len(ss))
	for _, s := range ss {
		if s == "" {
			continue
		}
		id, err := kernel.ParseID(s)
		if err != nil {
			return nil, errors.Wrap(errors.KindParam, "iam.invalid_id", "dept_id 无效", err)
		}
		out = append(out, id)
	}
	return out, nil
}

// RegisterAdminRoutes mounts /admin/roles + registration-applications endpoints.
// Caller is expected to admin-gate this group via TenantAdminRequired.
func RegisterAdminRoutes(r *gin.RouterGroup, svc *Service, pool *pgxpool.Pool) {
	// Registration applications.
	// Tenant admins see only applications targeting their tenant; platform admins see all.
	r.GET("/admin/registrations", func(c *gin.Context) {
		claims, _ := ClaimsFromContext(c.Request.Context())
		status := c.DefaultQuery("status", AppStatusPending)
		if status == "all" {
			status = ""
		}
		var filter *kernel.ID
		if !svc.IsPlatformAdminUser(c.Request.Context(), claims.PlatformUserID) {
			tid := claims.TenantID
			filter = &tid
		}
		apps, err := svc.ListApplications(c.Request.Context(), status, filter)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"applications": apps, "count": len(apps)})
	})

	r.POST("/admin/registrations/:id/approve", func(c *gin.Context) {
		claims, _ := ClaimsFromContext(c.Request.Context())
		appID, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_id", "申请 ID 无效", err))
			return
		}
		var req struct {
			Role string `json:"role"` // tenant_member (default) or tenant_admin
		}
		// Body is optional (role defaults in the service); ignore bind errors.
		_ = c.ShouldBindJSON(&req)
		// Tenant admins can only approve applications targeting THEIR tenant.
		// Platform admins may approve any.
		app, err := svc.GetApplication(c.Request.Context(), appID)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		if app == nil {
			apiresp.Fail(c, errors.New(errors.KindNotFound, "iam.application_not_found", "申请不存在"))
			return
		}
		if !svc.IsPlatformAdminUser(c.Request.Context(), claims.PlatformUserID) {
			if app.TargetTenantID == nil || *app.TargetTenantID != claims.TenantID {
				apiresp.Fail(c, errors.New(errors.KindForbidden, "iam.cross_tenant_approval_forbidden",
					"租户管理员只能审批进入本租户的申请"))
				return
			}
		}
		u, err := svc.ApproveApplication(c.Request.Context(), pool, ApproveApplicationCmd{
			ApplicationID: appID,
			ReviewerID:    claims.PlatformUserID,
			Role:          req.Role,
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"user": u})
	})

	r.POST("/admin/registrations/:id/reject", func(c *gin.Context) {
		claims, _ := ClaimsFromContext(c.Request.Context())
		appID, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_id", "申请 ID 无效", err))
			return
		}
		var req struct {
			Reason string `json:"reason"`
		}
		_ = c.ShouldBindJSON(&req)
		// Same cross-tenant guard as approve: a tenant_admin may only act on
		// applications targeting their own tenant; platform admins may act on any.
		app, err := svc.GetApplication(c.Request.Context(), appID)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		if app == nil {
			apiresp.Fail(c, errors.New(errors.KindNotFound, "iam.application_not_found", "申请不存在"))
			return
		}
		if !svc.IsPlatformAdminUser(c.Request.Context(), claims.PlatformUserID) {
			if app.TargetTenantID == nil || *app.TargetTenantID != claims.TenantID {
				apiresp.Fail(c, errors.New(errors.KindForbidden, "iam.cross_tenant_reject_forbidden",
					"租户管理员只能处理进入本租户的申请"))
				return
			}
		}
		if err := svc.RejectApplication(c.Request.Context(), appID, claims.PlatformUserID, req.Reason); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

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
		var req struct {
			Code      string   `json:"code"`
			Name      string   `json:"name"`
			DataScope string   `json:"data_scope"`
			DeptIDs   []string `json:"dept_ids"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_request", "请求格式错误", err))
			return
		}
		deptIDs, err := parseIDs(req.DeptIDs)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		role, err := svc.CreateRole(c.Request.Context(), CreateRoleCmd{
			TenantID: tid, Code: req.Code, Name: req.Name,
			DataScope: req.DataScope, DeptIDs: deptIDs,
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.Created(c, role)
	})

	r.PATCH("/admin/roles/:id", func(c *gin.Context) {
		tid, _ := kernel.TenantIDFromContext(c.Request.Context())
		rid, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_id", "角色 ID 无效", err))
			return
		}
		var req struct {
			Code      *string   `json:"code"`
			Name      *string   `json:"name"`
			DataScope *string   `json:"data_scope"`
			DeptIDs   *[]string `json:"dept_ids"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_request", "请求格式错误", err))
			return
		}
		cmd := UpdateRoleCmd{TenantID: tid, RoleID: rid, Code: req.Code, Name: req.Name, DataScope: req.DataScope}
		if req.DeptIDs != nil {
			ids, err := parseIDs(*req.DeptIDs)
			if err != nil {
				apiresp.Fail(c, err)
				return
			}
			cmd.DeptIDs = &ids
		}
		if err := svc.UpdateRole(c.Request.Context(), cmd); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
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

// RegisterPlatformAdminRoutes mounts the platform console API under /platform/*
// (platform admin only — caller gates with PlatformAdminRequired; this group has
// NO tenant context). Covers cross-tenant user management, registration review,
// and platform stats.
func RegisterPlatformAdminRoutes(r *gin.RouterGroup, svc *Service, pool *pgxpool.Pool) {
	// --- Platform overview stats ---
	r.GET("/platform/stats", func(c *gin.Context) {
		ctx := c.Request.Context()
		var orgs, users, pending int
		_ = pool.QueryRow(ctx, `SELECT count(*) FROM public.tenant WHERE status='active'`).Scan(&orgs)
		_ = pool.QueryRow(ctx, `SELECT count(*) FROM public.platform_user`).Scan(&users)
		_ = pool.QueryRow(ctx, `SELECT count(*) FROM public.registration_application WHERE status='pending'`).Scan(&pending)
		apiresp.OK(c, gin.H{"organizations": orgs, "users": users, "pending_registrations": pending})
	})

	// --- Cross-tenant registration review (platform admin sees ALL) ---
	r.GET("/platform/registrations", func(c *gin.Context) {
		status := c.DefaultQuery("status", AppStatusPending)
		if status == "all" {
			status = ""
		}
		apps, err := svc.ListApplications(c.Request.Context(), status, nil) // nil filter = all tenants
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"applications": apps, "count": len(apps)})
	})
	r.POST("/platform/registrations/:id/approve", func(c *gin.Context) {
		claims, _ := ClaimsFromContext(c.Request.Context())
		appID, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_id", "申请 ID 无效", err))
			return
		}
		var req struct {
			Role string `json:"role"`
		}
		_ = c.ShouldBindJSON(&req)
		u, err := svc.ApproveApplication(c.Request.Context(), pool, ApproveApplicationCmd{
			ApplicationID: appID, ReviewerID: claims.PlatformUserID, Role: req.Role,
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"user": u})
	})
	r.POST("/platform/registrations/:id/reject", func(c *gin.Context) {
		claims, _ := ClaimsFromContext(c.Request.Context())
		appID, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_id", "申请 ID 无效", err))
			return
		}
		var req struct {
			Reason string `json:"reason"`
		}
		_ = c.ShouldBindJSON(&req)
		if err := svc.RejectApplication(c.Request.Context(), appID, claims.PlatformUserID, req.Reason); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	// --- Cross-tenant platform users ---
	r.GET("/platform/users", func(c *gin.Context) {
		users, err := svc.ListUsers(c.Request.Context(), c.Query("search"), 200)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"users": users})
	})

	r.POST("/platform/users", func(c *gin.Context) {
		var req struct {
			Username       string `json:"username" binding:"required"`
			RealName       string `json:"real_name" binding:"required"`
			Phone          string `json:"phone"`
			Password       string `json:"password" binding:"required"`
			OrganizationID string `json:"organization_id" binding:"required"`
			Role           string `json:"role"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_request", "请求格式错误", err))
			return
		}
		oid, err := kernel.ParseID(req.OrganizationID)
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_organization_id", "organization_id 无效", err))
			return
		}
		u, err := svc.AdminCreateUser(c.Request.Context(), pool, CreateUserByAdminCmd{
			Username: req.Username, RealName: req.RealName,
			Phone: req.Phone, Password: req.Password,
			OrganizationID: oid, Role: req.Role,
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.Created(c, u)
	})

	r.GET("/platform/users/:id", func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_id", "id 无效", err))
			return
		}
		u, err := svc.GetUser(c.Request.Context(), id)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		if u == nil {
			apiresp.Fail(c, errors.New(errors.KindNotFound, "iam.user_not_found", "用户不存在"))
			return
		}
		apiresp.OK(c, u)
	})

	r.POST("/platform/users/:id/disable", func(c *gin.Context) {
		id, _ := kernel.ParseID(c.Param("id"))
		if err := svc.SetUserStatus(c.Request.Context(), id, "disabled"); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.POST("/platform/users/:id/enable", func(c *gin.Context) {
		id, _ := kernel.ParseID(c.Param("id"))
		if err := svc.SetUserStatus(c.Request.Context(), id, "active"); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.POST("/platform/users/:id/reset-password", func(c *gin.Context) {
		id, _ := kernel.ParseID(c.Param("id"))
		var req struct {
			NewPassword string `json:"new_password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_request", "请求格式错误", err))
			return
		}
		if err := svc.AdminResetPassword(c.Request.Context(), id, req.NewPassword); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
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
		isPA := svc.IsPlatformAdminUser(c.Request.Context(), claims.PlatformUserID)
		apiresp.OK(c, gin.H{
			"is_tenant_admin":   isTA,
			"is_platform_admin": isPA,
			"has_tenant":        claims.TenantID != "",
		})
	})

	// Online users (在线用户): active sessions bound to the current tenant.
	r.GET("/admin/online", func(c *gin.Context) {
		tc, ok := tenantdb.FromContext(c.Request.Context())
		if !ok || tc.ID == "" {
			apiresp.Fail(c, errors.New(errors.KindForbidden, "iam.no_tenant", "缺少租户上下文"))
			return
		}
		sessions, err := svc.ListOnlineSessions(c.Request.Context(), kernel.ID(tc.ID), tc.SchemaName)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"sessions": sessions})
	})

	// Kick (force-logout) a session — scoped to the acting admin's tenant so a
	// tenant admin cannot revoke sessions outside their own organization.
	r.POST("/admin/online/:sid/kick", func(c *gin.Context) {
		tc, ok := tenantdb.FromContext(c.Request.Context())
		if !ok || tc.ID == "" {
			apiresp.Fail(c, errors.New(errors.KindForbidden, "iam.no_tenant", "缺少租户上下文"))
			return
		}
		sid, err := kernel.ParseID(c.Param("sid"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_session_id", "会话 ID 无效", err))
			return
		}
		owner, err := svc.GetSessionTenant(c.Request.Context(), sid)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		if owner == "" || owner != kernel.ID(tc.ID) {
			apiresp.Fail(c, errors.New(errors.KindForbidden, "iam.session_not_in_tenant", "无权操作该会话"))
			return
		}
		if err := svc.Logout(c.Request.Context(), sid); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
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

// PlatformAdminRequired gates the platform console. It checks the GLOBAL
// is_platform_admin flag on the authenticated platform_user and does NOT require
// any tenant context — platform admins govern across all tenants and need not be
// a member of any tenant.
func PlatformAdminRequired(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := ClaimsFromContext(c.Request.Context())
		if !ok || claims.PlatformUserID == "" {
			apiresp.Fail(c, errors.New(errors.KindForbidden, "iam.platform_admin_required", "请使用平台管理员账号"))
			return
		}
		if !svc.IsPlatformAdminUser(c.Request.Context(), claims.PlatformUserID) {
			apiresp.Fail(c, errors.New(errors.KindForbidden, "iam.platform_admin_required", "需要平台管理员权限"))
			return
		}
		c.Next()
	}
}
