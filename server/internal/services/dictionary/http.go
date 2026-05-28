package dictionary

import (
	"github.com/gin-gonic/gin"
	"github.com/leo/iop/server/internal/interface/apiresp"
)

// RegisterRoutes wires GET /dict/:typeCode to the engine.
// Called by app/wiring.go after engine creation.
func RegisterRoutes(r *gin.RouterGroup, svc *Service) {
	r.GET("/dict/:typeCode", func(c *gin.Context) {
		items, err := svc.Lookup(c.Request.Context(), c.Param("typeCode"))
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"items": items})
	})
}
