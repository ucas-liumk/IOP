package appstore

import (
	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/services/iam"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/module"
)

// RegisterMeRoutes mounts /me/apps under the auth-only group.
// Returns just the apps this tenant has installed.
func RegisterMeRoutes(r *gin.RouterGroup, svc *Service) {
	r.GET("/me/apps", func(c *gin.Context) {
		tid, _ := kernel.TenantIDFromContext(c.Request.Context())
		if tid == "" {
			apiresp.OK(c, gin.H{"apps": []module.Manifest{}})
			return
		}
		apps, err := svc.MyApps(c.Request.Context(), tid)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"apps": apps})
	})
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
func RegisterAdminRoutes(r *gin.RouterGroup, svc *Service) {
	r.POST("/admin/apps/:code/install", func(c *gin.Context) {
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

	r.DELETE("/admin/apps/:code/install", func(c *gin.Context) {
		tid, _ := kernel.TenantIDFromContext(c.Request.Context())
		code := c.Param("code")
		if err := svc.Uninstall(c.Request.Context(), tid, code); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true, "uninstalled": code})
	})
}
