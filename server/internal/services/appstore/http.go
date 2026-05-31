package appstore

import (
	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/services/iam"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/module"
)

type AuthzFunc func(resource, action string) gin.HandlerFunc

// RegisterMeRoutes mounts /me/apps under the auth+tenant group (authT).
// Any logged-in member can read AND curate their own per-user workspace:
//   - GET    /me/apps          ordered manifests for the current user
//   - POST   /me/apps/:code    add an app to the workspace (append)
//   - DELETE /me/apps/:code    remove an app from the workspace
//   - PUT    /me/apps/order    rewrite ordering, body {"codes":[...]}
//
// No admin gate. Tenant + platform_user come from the verified claims; with no
// tenant context the read returns an empty list and writes 400 gracefully.
func RegisterMeRoutes(r *gin.RouterGroup, svc *Service) {
	r.GET("/me/apps", func(c *gin.Context) {
		ctx := c.Request.Context()
		tid, _ := kernel.TenantIDFromContext(ctx)
		puid, _ := kernel.PlatformUserIDFromContext(ctx)
		if tid == "" || puid == "" {
			apiresp.OK(c, gin.H{"apps": []module.Manifest{}})
			return
		}
		codes, err := svc.MyApps(ctx, puid, tid)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"apps": svc.ManifestsFor(codes)})
	})

	r.POST("/me/apps/:code", func(c *gin.Context) {
		ctx := c.Request.Context()
		tid, puid, ok := currentUserTenant(c)
		if !ok {
			return
		}
		code := c.Param("code")
		if code == "" {
			apiresp.Fail(c, errors.New(errors.KindParam, "appstore.missing_code", "缺少应用 code"))
			return
		}
		if err := svc.AddUserApp(ctx, puid, tid, code); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true, "added": code})
	})

	r.DELETE("/me/apps/:code", func(c *gin.Context) {
		ctx := c.Request.Context()
		tid, puid, ok := currentUserTenant(c)
		if !ok {
			return
		}
		code := c.Param("code")
		if err := svc.RemoveUserApp(ctx, puid, tid, code); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true, "removed": code})
	})

	r.PUT("/me/apps/order", func(c *gin.Context) {
		ctx := c.Request.Context()
		tid, puid, ok := currentUserTenant(c)
		if !ok {
			return
		}
		var req struct {
			Codes []string `json:"codes"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.New(errors.KindParam, "appstore.bad_body", "请求体格式错误"))
			return
		}
		if err := svc.SetUserAppOrder(ctx, puid, tid, req.Codes); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})
}

// currentUserTenant resolves the (tenant, platform_user) for a write to the
// per-user workspace. It fails the request with a 400 and returns ok=false if
// there is no tenant context, so the caller can simply `return`.
func currentUserTenant(c *gin.Context) (tid, puid kernel.ID, ok bool) {
	ctx := c.Request.Context()
	tid, _ = kernel.TenantIDFromContext(ctx)
	puid, _ = kernel.PlatformUserIDFromContext(ctx)
	if tid == "" || puid == "" {
		apiresp.Fail(c, errors.New(errors.KindParam, "appstore.no_tenant", "请先选择组织"))
		return "", "", false
	}
	return tid, puid, true
}

// RegisterCatalogRoutes mounts /apps/catalog under the auth+tenant group.
// Returns every module the binary supports with an installed flag.
func RegisterCatalogRoutes(r *gin.RouterGroup, svc *Service) {
	r.GET("/apps/catalog", func(c *gin.Context) {
		tid, _ := kernel.TenantIDFromContext(c.Request.Context())
		entries, err := svc.Catalog(c.Request.Context(), tid)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"apps": entries})
	})
}

// RegisterAdminRoutes mounts /admin/apps/:code/install (POST + DELETE).
// Caller is expected to wrap with TenantAdminRequired.
func RegisterAdminRoutes(r *gin.RouterGroup, svc *Service, authz AuthzFunc) {
	r.POST("/admin/apps/:code/install", authz("app", "write"), func(c *gin.Context) {
		tid, _ := kernel.TenantIDFromContext(c.Request.Context())
		claims, _ := iam.ClaimsFromContext(c.Request.Context())
		code := c.Param("code")
		if code == "" {
			apiresp.Fail(c, errors.New(errors.KindParam, "appstore.missing_code", "缺少应用 code"))
			return
		}
		if err := svc.Install(c.Request.Context(), tid, code, claims.PlatformUserID); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true, "installed": code})
	})

	r.DELETE("/admin/apps/:code/install", authz("app", "write"), func(c *gin.Context) {
		tid, _ := kernel.TenantIDFromContext(c.Request.Context())
		code := c.Param("code")
		if err := svc.Uninstall(c.Request.Context(), tid, code); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true, "uninstalled": code})
	})
}
