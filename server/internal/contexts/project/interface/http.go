// Package iface wires the project module's REST routes. Mounted under
// /api/apps/project/* by the module Registry; every route is RBAC-gated via the
// module.AuthzFunc passed in (resource×action from the Manifest).
package iface

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/contexts/project/application"
	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/services/iam"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/module"
)

const projectRes = "project"

func RegisterRoutes(r *gin.RouterGroup, svc *application.Service, authz module.AuthzFunc) {
	gate := func(action string) gin.HandlerFunc {
		if authz == nil {
			return func(c *gin.Context) { c.Next() }
		}
		return authz(projectRes, action)
	}
	member := func(c *gin.Context) kernel.ID {
		claims, _ := iam.ClaimsFromContext(c.Request.Context())
		return claims.MemberID
	}

	// ---- Projects ----
	r.GET("/projects", gate("read"), func(c *gin.Context) {
		ps, err := svc.ListProjects(c.Request.Context())
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"projects": ps})
	})

	r.POST("/projects", gate("write"), func(c *gin.Context) {
		var req struct {
			Name        string `json:"name" binding:"required"`
			Description string `json:"description"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "project.invalid_request", "请求格式错误", err))
			return
		}
		p, err := svc.CreateProject(c.Request.Context(), application.CreateProjectCmd{
			CreatedBy: member(c), Name: req.Name, Description: req.Description,
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.Created(c, p)
	})

	r.GET("/projects/:id", gate("read"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "project.invalid_id", "id 无效", err))
			return
		}
		p, err := svc.GetBoard(c.Request.Context(), id)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, p)
	})

	r.PATCH("/projects/:id", gate("write"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "project.invalid_id", "id 无效", err))
			return
		}
		var req struct {
			Name        *string `json:"name"`
			Description *string `json:"description"`
			Status      *string `json:"status"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "project.invalid_request", "请求格式错误", err))
			return
		}
		p, err := svc.UpdateProject(c.Request.Context(), application.UpdateProjectCmd{
			ID: id, Name: req.Name, Description: req.Description, Status: req.Status,
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, p)
	})

	r.DELETE("/projects/:id", gate("write"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "project.invalid_id", "id 无效", err))
			return
		}
		if err := svc.DeleteProject(c.Request.Context(), id); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	// ---- Columns ----
	r.POST("/projects/:id/columns", gate("write"), func(c *gin.Context) {
		pid, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "project.invalid_id", "id 无效", err))
			return
		}
		var req struct {
			Name string `json:"name" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "project.invalid_request", "请求格式错误", err))
			return
		}
		col, err := svc.CreateColumn(c.Request.Context(), application.CreateColumnCmd{
			ProjectID: pid, Name: req.Name,
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.Created(c, col)
	})

	r.PATCH("/columns/:id", gate("write"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "project.invalid_id", "id 无效", err))
			return
		}
		var req struct {
			Name     *string `json:"name"`
			OrderNum *int    `json:"order_num"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "project.invalid_request", "请求格式错误", err))
			return
		}
		col, err := svc.UpdateColumn(c.Request.Context(), application.UpdateColumnCmd{
			ID: id, Name: req.Name, OrderNum: req.OrderNum,
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, col)
	})

	r.DELETE("/columns/:id", gate("write"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "project.invalid_id", "id 无效", err))
			return
		}
		if err := svc.DeleteColumn(c.Request.Context(), id); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	// ---- Cards ----
	r.POST("/projects/:id/cards", gate("write"), func(c *gin.Context) {
		pid, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "project.invalid_id", "id 无效", err))
			return
		}
		var req struct {
			ColumnID    string `json:"column_id" binding:"required"`
			Title       string `json:"title" binding:"required"`
			Description string `json:"description"`
			AssigneeID  string `json:"assignee_id"`
			DueDate     string `json:"due_date"`
			Priority    int    `json:"priority"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "project.invalid_request", "请求格式错误", err))
			return
		}
		colID, err := kernel.ParseID(req.ColumnID)
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "project.invalid_id", "column_id 无效", err))
			return
		}
		cmd := application.CreateCardCmd{
			ProjectID: pid, ColumnID: colID, Title: req.Title,
			Description: req.Description, Priority: req.Priority,
		}
		cmd.AssigneeID = optID(req.AssigneeID)
		cmd.DueDate = optTime(req.DueDate)
		card, err := svc.CreateCard(c.Request.Context(), cmd)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.Created(c, card)
	})

	r.GET("/cards/:id", gate("read"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "project.invalid_id", "id 无效", err))
			return
		}
		card, err := svc.GetCard(c.Request.Context(), id)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, card)
	})

	r.PATCH("/cards/:id", gate("write"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "project.invalid_id", "id 无效", err))
			return
		}
		var req struct {
			Title       *string `json:"title"`
			Description *string `json:"description"`
			Priority    *int    `json:"priority"`
			AssigneeID  *string `json:"assignee_id"` // "" clears
			DueDate     *string `json:"due_date"`    // "" clears
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "project.invalid_request", "请求格式错误", err))
			return
		}
		cmd := application.UpdateCardCmd{
			ID: id, Title: req.Title, Description: req.Description, Priority: req.Priority,
		}
		if req.AssigneeID != nil {
			if *req.AssigneeID == "" {
				cmd.ClearAssignee = true
			} else {
				cmd.AssigneeID = optID(*req.AssigneeID)
			}
		}
		if req.DueDate != nil {
			if *req.DueDate == "" {
				cmd.ClearDue = true
			} else {
				cmd.DueDate = optTime(*req.DueDate)
			}
		}
		card, err := svc.UpdateCard(c.Request.Context(), cmd)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, card)
	})

	r.POST("/cards/:id/move", gate("write"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "project.invalid_id", "id 无效", err))
			return
		}
		var req struct {
			ColumnID string `json:"column_id" binding:"required"`
			OrderNum int    `json:"order_num"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "project.invalid_request", "请求格式错误", err))
			return
		}
		colID, err := kernel.ParseID(req.ColumnID)
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "project.invalid_id", "column_id 无效", err))
			return
		}
		card, err := svc.MoveCard(c.Request.Context(), application.MoveCmd{
			CardID: id, ColumnID: colID, OrderNum: req.OrderNum,
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, card)
	})

	r.DELETE("/cards/:id", gate("write"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "project.invalid_id", "id 无效", err))
			return
		}
		if err := svc.DeleteCard(c.Request.Context(), id); err != nil {
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

func optTime(s string) *time.Time {
	if s == "" {
		return nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return &t
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return &t
	}
	return nil
}
