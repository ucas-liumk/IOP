// Package iface wires the lcform module's REST routes. Mounted under
// /api/apps/lcform/* by the module Registry; every route is RBAC-gated via the
// module.AuthzFunc passed in (resource×action from the Manifest).
package iface

import (
	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/contexts/lcform/application"
	"github.com/leo/iop/server/internal/contexts/lcform/domain"
	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/services/iam"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/module"
)

const (
	defRes   = "lcform.def"   // manage
	entryRes = "lcform.entry" // read / write
)

func RegisterRoutes(r *gin.RouterGroup, svc *application.Service, authz module.AuthzFunc) {
	gate := func(resource, action string) gin.HandlerFunc {
		if authz == nil {
			return func(c *gin.Context) { c.Next() }
		}
		return authz(resource, action)
	}
	member := func(c *gin.Context) kernel.ID {
		claims, _ := iam.ClaimsFromContext(c.Request.Context())
		return claims.MemberID
	}

	// ---- Form definitions ----
	// Listing forms only needs entry:read (anyone who can fill a form can browse the center).
	r.GET("/forms", gate(entryRes, "read"), func(c *gin.Context) {
		defs, err := svc.ListDefs(c.Request.Context(), c.Query("include_archived") == "1")
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"forms": defs})
	})

	r.GET("/forms/:id", gate(entryRes, "read"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "lcform.invalid_id", "id 无效", err))
			return
		}
		d, err := svc.GetDef(c.Request.Context(), id)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, d)
	})

	r.POST("/forms", gate(defRes, "manage"), func(c *gin.Context) {
		var req defReq
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "lcform.invalid_request", "请求格式错误", err))
			return
		}
		d, err := svc.CreateDef(c.Request.Context(), req.toCmd(member(c)))
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.Created(c, d)
	})

	r.PUT("/forms/:id", gate(defRes, "manage"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "lcform.invalid_id", "id 无效", err))
			return
		}
		var req defReq
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "lcform.invalid_request", "请求格式错误", err))
			return
		}
		d, err := svc.UpdateDef(c.Request.Context(), id, req.toCmd(member(c)))
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, d)
	})

	r.DELETE("/forms/:id", gate(defRes, "manage"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "lcform.invalid_id", "id 无效", err))
			return
		}
		if err := svc.DeleteDef(c.Request.Context(), id); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	// ---- Entries ----
	r.POST("/forms/:id/entries", gate(entryRes, "write"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "lcform.invalid_id", "id 无效", err))
			return
		}
		var req struct {
			Data map[string]any `json:"data"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "lcform.invalid_request", "请求格式错误", err))
			return
		}
		if req.Data == nil {
			req.Data = map[string]any{}
		}
		e, err := svc.SubmitEntry(c.Request.Context(), id, member(c), req.Data)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.Created(c, e)
	})

	r.GET("/forms/:id/entries", gate(entryRes, "read"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "lcform.invalid_id", "id 无效", err))
			return
		}
		var pg kernel.Pagination
		_ = c.ShouldBindQuery(&pg)
		page, err := svc.ListEntries(c.Request.Context(), id, c.Query("search"), pg)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, page)
	})

	r.GET("/forms/:id/entries/export", gate(entryRes, "read"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "lcform.invalid_id", "id 无效", err))
			return
		}
		filename, rows, err := svc.ExportCSV(c.Request.Context(), id)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.CSV(c, filename, rows)
	})
}

// defReq is the create/update payload for a form definition.
type defReq struct {
	Code   string         `json:"code"`
	Name   string         `json:"name" binding:"required"`
	Icon   string         `json:"icon"`
	Status string         `json:"status"`
	Fields []domain.Field `json:"fields"`
}

func (r defReq) toCmd(by kernel.ID) application.SaveDefCmd {
	return application.SaveDefCmd{
		Code: r.Code, Name: r.Name, Icon: r.Icon, Status: r.Status,
		Fields: r.Fields, CreatedBy: by,
	}
}
