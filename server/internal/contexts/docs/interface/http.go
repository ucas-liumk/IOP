// Package iface wires the docs (知识库) module's REST routes. Mounted under
// /api/apps/docs/* by the module Registry; every route is RBAC-gated via the
// module.AuthzFunc passed in (resource×action from the Manifest).
package iface

import (
	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/contexts/docs/application"
	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/services/iam"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/module"
)

const docsRes = "docs.node"

func RegisterRoutes(r *gin.RouterGroup, svc *application.Service, authz module.AuthzFunc) {
	gate := func(resource, action string) gin.HandlerFunc {
		if authz == nil {
			return func(c *gin.Context) { c.Next() }
		}
		return authz(resource, action)
	}
	actor := func(c *gin.Context) kernel.ID {
		claims, _ := iam.ClaimsFromContext(c.Request.Context())
		return claims.MemberID
	}

	// GET /tree — full folder/doc tree (metadata only).
	r.GET("/tree", gate(docsRes, "read"), func(c *gin.Context) {
		tree, err := svc.Tree(c.Request.Context())
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"tree": tree})
	})

	// GET /docs/:id — a single node with its content.
	r.GET("/docs/:id", gate(docsRes, "read"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "docs.invalid_id", "id 无效", err))
			return
		}
		n, err := svc.Get(c.Request.Context(), id)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, n)
	})

	// POST /docs — create a folder or doc.
	r.POST("/docs", gate(docsRes, "write"), func(c *gin.Context) {
		var req struct {
			ParentID string `json:"parent_id"`
			Title    string `json:"title" binding:"required"`
			Type     string `json:"type"` // folder / doc (default doc)
			Content  string `json:"content"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "docs.invalid_request", "请求格式错误", err))
			return
		}
		n, err := svc.Create(c.Request.Context(), application.CreateCmd{
			Actor: actor(c), ParentID: optID(req.ParentID), Title: req.Title,
			Type: req.Type, Content: req.Content,
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.Created(c, n)
	})

	// PUT /docs/:id — rename and/or save content. A nil title leaves the title
	// untouched; content is always written (use the current content to no-op).
	r.PUT("/docs/:id", gate(docsRes, "write"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "docs.invalid_id", "id 无效", err))
			return
		}
		var req struct {
			Title   *string `json:"title"`
			Content *string `json:"content"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "docs.invalid_request", "请求格式错误", err))
			return
		}
		// If only a title was supplied (no content), treat it as a rename.
		if req.Content == nil {
			if req.Title == nil {
				apiresp.Fail(c, errors.New(errors.KindParam, "docs.nothing_to_save", "没有要保存的内容"))
				return
			}
			if err := svc.Rename(c.Request.Context(), actor(c), id, *req.Title); err != nil {
				apiresp.Fail(c, err)
				return
			}
			n, err := svc.Get(c.Request.Context(), id)
			if err != nil {
				apiresp.Fail(c, err)
				return
			}
			apiresp.OK(c, n)
			return
		}
		n, err := svc.SaveDoc(c.Request.Context(), actor(c), id, req.Title, *req.Content)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, n)
	})

	// POST /docs/:id/move — reparent / reorder.
	r.POST("/docs/:id/move", gate(docsRes, "write"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "docs.invalid_id", "id 无效", err))
			return
		}
		var req struct {
			ParentID string `json:"parent_id"` // "" => move to root
			OrderNum int    `json:"order_num"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "docs.invalid_request", "请求格式错误", err))
			return
		}
		cmd := application.MoveCmd{Actor: actor(c), ID: id, OrderNum: req.OrderNum}
		if req.ParentID == "" {
			cmd.ToRoot = true
		} else {
			cmd.ParentID = optID(req.ParentID)
		}
		if err := svc.Move(c.Request.Context(), cmd); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	// DELETE /docs/:id — delete a node (and its subtree, via CASCADE).
	r.DELETE("/docs/:id", gate(docsRes, "delete"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "docs.invalid_id", "id 无效", err))
			return
		}
		if err := svc.Delete(c.Request.Context(), id); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})
}

func optID(s string) *kernel.ID {
	if s == "" {
		return nil
	}
	id, err := kernel.ParseID(s)
	if err != nil {
		return nil
	}
	return &id
}
