package iface

import (
	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/contexts/crm/application"
	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/shared/errors"
)

// RegisterRoutes mounts /items under the caller's group (typically /api/apps/crm).
func RegisterRoutes(r *gin.RouterGroup, svc *application.Service) {
	r.GET("/items", func(c *gin.Context) {
		apiresp.OK(c, gin.H{"items": []any{}, "message": "crm module bootstrapped"})
	})
	r.POST("/items", func(c *gin.Context) {
		var req struct{ Title, Body string }
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "crm.invalid_request", "请求格式错误", err))
			return
		}
		item, err := svc.CreateItem(c.Request.Context(), req.Title, req.Body)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.Created(c, item)
	})
}
