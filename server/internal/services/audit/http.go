package audit

import (
	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/shared/kernel"
)

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
