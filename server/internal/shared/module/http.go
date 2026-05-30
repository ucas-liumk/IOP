package module

import (
	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/interface/apiresp"
)

// RegisterAdminRoutes mounts /admin/permissions — used by the role-editor UI
// to surface known resources × actions as dropdown options.
func RegisterAdminRoutes(r *gin.RouterGroup, registry *Registry) {
	r.GET("/admin/permissions", func(c *gin.Context) {
		// Group by resource for cleaner UI consumption.
		byResource := map[string][]Permission{}
		for _, p := range registry.AllPermissions() {
			byResource[p.Resource] = append(byResource[p.Resource], p)
		}
		apiresp.OK(c, gin.H{
			"permissions": registry.AllPermissions(),
			"by_resource": byResource,
		})
	})
}
