// Package iface wires the mindmap module's REST routes. Mounted under
// /api/apps/mindmap/* by the module Registry; every route is RBAC-gated via the
// module.AuthzFunc passed in (resource×action from the Manifest).
package iface

import (
	"encoding/json"

	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/contexts/mindmap/application"
	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/services/iam"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/module"
)

const mapRes = "mindmap.map"

func RegisterRoutes(r *gin.RouterGroup, svc *application.Service, authz module.AuthzFunc) {
	gate := func(resource, action string) gin.HandlerFunc {
		if authz == nil {
			return func(c *gin.Context) { c.Next() }
		}
		return authz(resource, action)
	}
	owner := func(c *gin.Context) kernel.ID {
		claims, _ := iam.ClaimsFromContext(c.Request.Context())
		return claims.MemberID
	}

	r.GET("/maps", gate(mapRes, "read"), func(c *gin.Context) {
		maps, err := svc.List(c.Request.Context(), owner(c))
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"maps": maps})
	})

	r.POST("/maps", gate(mapRes, "write"), func(c *gin.Context) {
		var req struct {
			Title string          `json:"title" binding:"required"`
			Data  json.RawMessage `json:"data"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "mindmap.invalid_request", "请求格式错误", err))
			return
		}
		m, err := svc.Create(c.Request.Context(), application.CreateCmd{
			Owner: owner(c), Title: req.Title, Data: req.Data,
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.Created(c, m)
	})

	r.GET("/maps/:id", gate(mapRes, "read"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "mindmap.invalid_id", "id 无效", err))
			return
		}
		m, err := svc.Get(c.Request.Context(), owner(c), id)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, m)
	})

	r.PUT("/maps/:id", gate(mapRes, "write"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "mindmap.invalid_id", "id 无效", err))
			return
		}
		var req struct {
			Title *string          `json:"title"`
			Data  *json.RawMessage `json:"data"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "mindmap.invalid_request", "请求格式错误", err))
			return
		}
		m, err := svc.Update(c.Request.Context(), application.UpdateCmd{
			Owner: owner(c), ID: id, Title: req.Title, Data: req.Data,
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, m)
	})

	r.DELETE("/maps/:id", gate(mapRes, "delete"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "mindmap.invalid_id", "id 无效", err))
			return
		}
		if err := svc.Delete(c.Request.Context(), owner(c), id); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})
}
