package iam

import (
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/services/audit"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
)

// PlatformAccess gates the platform console: any platform role (or legacy
// is_platform_admin) may enter. Replaces PlatformAdminRequired on the platform group.
func PlatformAccess(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := ClaimsFromContext(c.Request.Context())
		if !ok || claims.PlatformUserID == "" {
			apiresp.Fail(c, errors.New(errors.KindForbidden, "iam.platform_access_required", "请使用平台账号"))
			return
		}
		if !svc.HasAnyPlatformRole(c.Request.Context(), claims.PlatformUserID) {
			apiresp.Fail(c, errors.New(errors.KindForbidden, "iam.platform_access_required", "无平台访问权限"))
			return
		}
		c.Next()
	}
}

// PlatformAuthz enforces a single (resource, action) on the platform side via the
// generic RBAC policy match, and records a platform audit entry on successful non-GET
// requests.
func PlatformAuthz(svc *Service, aud *audit.Service, resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := ClaimsFromContext(c.Request.Context())
		if !ok || claims.PlatformUserID == "" {
			apiresp.Fail(c, errors.New(errors.KindAuth, "iam.no_session", "未登录"))
			return
		}
		ctx := c.Request.Context()
		if err := svc.EnforcePlatform(ctx, claims.PlatformUserID, resource, action); err != nil {
			apiresp.Fail(c, err)
			return
		}

		c.Next()

		// Record successful writes only: apiresp.Fail aborts the context, so a
		// handler error (4xx/5xx) leaves IsAborted() true and is not audited here.
		if c.Request.Method != "GET" && !c.IsAborted() {
			roles, _ := svc.repo.ListPlatformRolesForUser(ctx, claims.PlatformUserID)
			codes := make([]string, 0, len(roles))
			for _, r := range roles {
				codes = append(codes, r.Code)
			}
			detail, _ := json.Marshal(gin.H{"path": c.Request.URL.Path, "status": c.Writer.Status()})
			aud.RecordPlatform(ctx, audit.PlatformEntry{
				Actor:      string(claims.PlatformUserID),
				ActorRole:  strings.Join(codes, ","),
				Action:     resource + "/" + action,
				Resource:   resource,
				ResourceID: c.Param("id"),
				TraceID:    kernel.TraceIDFromContext(ctx),
				Detail:     detail,
			})
		}
	}
}

// RegisterPlatformRBACRoutes mounts /platform/rbac/* on the platform group.
func RegisterPlatformRBACRoutes(r *gin.RouterGroup, svc *Service, aud *audit.Service) {
	// Current user's platform roles + permissions (front-end gating). Login-only.
	r.GET("/platform/rbac/me", func(c *gin.Context) {
		claims, _ := ClaimsFromContext(c.Request.Context())
		ctx := c.Request.Context()
		roles, _ := svc.repo.ListPlatformRolesForUser(ctx, claims.PlatformUserID)
		codes := []string{}
		isSuper := false
		for _, role := range roles {
			codes = append(codes, role.Code)
			if role.Code == "super_admin" {
				isSuper = true
			}
		}
		perms, _ := svc.PlatformPermissionsForUser(ctx, claims.PlatformUserID)
		apiresp.OK(c, gin.H{
			"roles": codes, "permissions": perms, "is_super_admin": isSuper,
		})
	})

	r.GET("/platform/rbac/roles", PlatformAuthz(svc, aud, "role", "manage"), func(c *gin.Context) {
		roles, err := svc.ListPlatformRolesWithPolicies(c.Request.Context())
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"roles": roles})
	})

	r.POST("/platform/rbac/roles", PlatformAuthz(svc, aud, "role", "manage"), func(c *gin.Context) {
		var req struct {
			Code string `json:"code" binding:"required"`
			Name string `json:"name" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_request", "请求格式错误", err))
			return
		}
		if err := svc.CreatePlatformRole(c.Request.Context(), req.Code, req.Name); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.DELETE("/platform/rbac/roles/:id", PlatformAuthz(svc, aud, "role", "manage"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_id", "id 无效", err))
			return
		}
		if err := svc.DeletePlatformRole(c.Request.Context(), id); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.POST("/platform/rbac/roles/:id/policies", PlatformAuthz(svc, aud, "role", "manage"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_id", "id 无效", err))
			return
		}
		var req struct {
			Resource string `json:"resource" binding:"required"`
			Action   string `json:"action" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_request", "请求格式错误", err))
			return
		}
		if err := svc.repo.AddPlatformPolicy(c.Request.Context(), id, req.Resource, req.Action); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.DELETE("/platform/rbac/roles/:id/policies", PlatformAuthz(svc, aud, "role", "manage"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_id", "id 无效", err))
			return
		}
		resource := c.Query("resource")
		action := c.Query("action")
		if resource == "" || action == "" {
			apiresp.Fail(c, errors.New(errors.KindParam, "iam.invalid_request", "resource 和 action 必填"))
			return
		}
		if err := svc.repo.RemovePlatformPolicy(c.Request.Context(), id, resource, action); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	// Batch policy edit (platform role): apply adds + removes in one transaction.
	// Coexists with the single add/remove routes above.
	r.POST("/platform/rbac/roles/:id/policies/batch", PlatformAuthz(svc, aud, "role", "manage"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_id", "id 无效", err))
			return
		}
		var req struct {
			Add    []PolicyChange `json:"add"`
			Remove []PolicyChange `json:"remove"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_request", "请求格式错误", err))
			return
		}
		if err := svc.BatchPlatformPolicy(c.Request.Context(), id, req.Add, req.Remove); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.POST("/platform/rbac/roles/:id/members", PlatformAuthz(svc, aud, "authz", "grant"), func(c *gin.Context) {
		claims, _ := ClaimsFromContext(c.Request.Context())
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_id", "id 无效", err))
			return
		}
		var req struct {
			PlatformUserID string `json:"platform_user_id" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_request", "请求格式错误", err))
			return
		}
		uid, err := kernel.ParseID(req.PlatformUserID)
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_id", "platform_user_id 无效", err))
			return
		}
		role, err := svc.repo.GetPlatformRoleByID(c.Request.Context(), id)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		if role == nil {
			apiresp.Fail(c, errors.New(errors.KindNotFound, "iam.role_not_found", "平台角色不存在"))
			return
		}
		if role.Code == "super_admin" && !svc.UserHasPlatformRole(c.Request.Context(), claims.PlatformUserID, "super_admin") {
			apiresp.Fail(c, errors.New(errors.KindForbidden, "iam.super_grant_forbidden", "仅超级管理员可授予 super_admin 角色"))
			return
		}
		if err := svc.repo.GrantPlatformRole(c.Request.Context(), id, uid, claims.PlatformUserID); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.DELETE("/platform/rbac/roles/:id/members/:uid", PlatformAuthz(svc, aud, "authz", "grant"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_id", "id 无效", err))
			return
		}
		uid, err := kernel.ParseID(c.Param("uid"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_id", "uid 无效", err))
			return
		}
		if err := svc.repo.RevokePlatformRole(c.Request.Context(), id, uid); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})
}
