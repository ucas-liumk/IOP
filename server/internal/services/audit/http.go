package audit

import (
	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/shared/kernel"
)

type AuthzFunc func(resource, action string) gin.HandlerFunc

// RegisterRoutes mounts /audit endpoints. Caller must wrap with auth + RBAC.
func RegisterRoutes(r *gin.RouterGroup, svc *Service) {
	r.GET("/audit/logs", func(c *gin.Context) {
		var p kernel.Pagination
		_ = c.ShouldBindQuery(&p)
		entries, err := svc.ListByTenant(c.Request.Context(), p)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"entries": entries})
	})
}

// loginActionLike matches the family of IAM login event topics
// (iam.user_logged_in / iam.user_logged_out / iam.login_failed and any future
// iam.*login* topic). Mirror this if new login topics are added.
const loginActionLike = "iam.%log%"

// RegisterAdminRoutes mounts the tenant-console operation/login log endpoints
// under /admin/*. Caller must wrap with TenantLoader + TenantAdminRequired.
//
//   - GET /admin/operlogs  : full tenant audit_log, optional ?actor= ?action= ?from= ?to=
//   - GET /admin/loginlogs : audit_log filtered to login event topics
func RegisterAdminRoutes(r *gin.RouterGroup, svc *Service, authz AuthzFunc) {
	r.GET("/admin/operlogs", authz("log", "read"), func(c *gin.Context) {
		var p kernel.Pagination
		_ = c.ShouldBindQuery(&p)
		entries, err := svc.ListByTenantFiltered(c.Request.Context(), p, ListFilter{
			Actor:  c.Query("actor"),
			Action: c.Query("action"),
			From:   c.Query("from"),
			To:     c.Query("to"),
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"entries": entries})
	})

	r.GET("/admin/loginlogs", authz("log", "read"), func(c *gin.Context) {
		var p kernel.Pagination
		_ = c.ShouldBindQuery(&p)
		entries, err := svc.ListByTenantFiltered(c.Request.Context(), p, ListFilter{
			ActionLike: loginActionLike,
			Actor:      c.Query("actor"),
			From:       c.Query("from"),
			To:         c.Query("to"),
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"entries": entries})
	})
}
