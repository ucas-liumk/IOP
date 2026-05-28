package notification

import (
	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/services/iam"
	"github.com/leo/iop/server/internal/shared/kernel"
)

// RegisterRoutes mounts /notifications endpoints.
func RegisterRoutes(r *gin.RouterGroup, svc *Service) {
	r.GET("/notifications/unread", func(c *gin.Context) {
		claims, _ := iam.ClaimsFromContext(c.Request.Context())
		var p kernel.Pagination
		_ = c.ShouldBindQuery(&p)
		notes, err := svc.ListUnread(c.Request.Context(), claims.MemberID, p)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"notifications": notes, "count": len(notes)})
	})

	r.POST("/notifications/:id/read", func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		if err := svc.MarkRead(c.Request.Context(), id); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})
}
