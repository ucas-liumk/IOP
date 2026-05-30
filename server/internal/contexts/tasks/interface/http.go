// Package iface wires the tasks module's REST routes. Mounted under
// /api/apps/tasks/* by the module Registry; every route is RBAC-gated via the
// module.AuthzFunc passed in (resource×action from the Manifest).
package iface

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/contexts/tasks/application"
	"github.com/leo/iop/server/internal/contexts/tasks/domain"
	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/services/iam"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/module"
)

const (
	listRes = "tasks.list"
	taskRes = "tasks.task"
)

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

	// ---- Lists ----
	r.GET("/lists", gate(listRes, "read"), func(c *gin.Context) {
		lists, err := svc.ListLists(c.Request.Context(), owner(c))
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"lists": lists})
	})

	r.POST("/lists", gate(listRes, "write"), func(c *gin.Context) {
		var req struct {
			Name  string `json:"name" binding:"required"`
			Color string `json:"color"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tasks.invalid_request", "请求格式错误", err))
			return
		}
		l, err := svc.CreateList(c.Request.Context(), application.CreateListCmd{
			Owner: owner(c), Name: req.Name, Color: req.Color,
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.Created(c, l)
	})

	r.PATCH("/lists/:id", gate(listRes, "write"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tasks.invalid_id", "id 无效", err))
			return
		}
		var req struct {
			Name      string `json:"name" binding:"required"`
			Color     string `json:"color"`
			SortOrder int    `json:"sort_order"`
			Archived  bool   `json:"archived"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tasks.invalid_request", "请求格式错误", err))
			return
		}
		if err := svc.UpdateList(c.Request.Context(), application.UpdateListCmd{
			Owner: owner(c), ID: id, Name: req.Name, Color: req.Color,
			SortOrder: req.SortOrder, Archived: req.Archived,
		}); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.DELETE("/lists/:id", gate(listRes, "delete"), func(c *gin.Context) {
		id, _ := kernel.ParseID(c.Param("id"))
		if err := svc.DeleteList(c.Request.Context(), owner(c), id); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	// ---- Tasks ----
	r.GET("/tasks", gate(taskRes, "read"), func(c *gin.Context) {
		f := domain.Filter{
			Owner:  owner(c),
			Status: c.Query("status"),
			View:   c.Query("view"),
			Tag:    c.Query("tag"),
		}
		if lid := c.Query("list_id"); lid != "" {
			if id, err := kernel.ParseID(lid); err == nil {
				f.ListID = &id
			}
		}
		tasks, err := svc.ListTasks(c.Request.Context(), f)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"tasks": tasks})
	})

	r.GET("/tasks/counts", gate(taskRes, "read"), func(c *gin.Context) {
		counts, err := svc.Counts(c.Request.Context(), owner(c))
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"counts": counts})
	})

	r.POST("/tasks", gate(taskRes, "write"), func(c *gin.Context) {
		var req struct {
			Title    string   `json:"title" binding:"required"`
			Note     string   `json:"note"`
			Priority int      `json:"priority"`
			ListID   string   `json:"list_id"`
			ParentID string   `json:"parent_id"`
			DueDate  string   `json:"due_date"`
			Tags     []string `json:"tags"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tasks.invalid_request", "请求格式错误", err))
			return
		}
		cmd := application.CreateTaskCmd{
			Owner: owner(c), Title: req.Title, Note: req.Note, Priority: req.Priority, Tags: req.Tags,
		}
		cmd.ListID = optID(req.ListID)
		cmd.ParentID = optID(req.ParentID)
		cmd.DueDate = optTime(req.DueDate)
		t, err := svc.CreateTask(c.Request.Context(), cmd)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.Created(c, t)
	})

	r.GET("/tasks/:id", gate(taskRes, "read"), func(c *gin.Context) {
		id, _ := kernel.ParseID(c.Param("id"))
		t, err := svc.GetTask(c.Request.Context(), owner(c), id)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		if t == nil {
			apiresp.Fail(c, errors.New(errors.KindNotFound, "tasks.task_not_found", "任务不存在"))
			return
		}
		apiresp.OK(c, t)
	})

	r.PATCH("/tasks/:id", gate(taskRes, "write"), func(c *gin.Context) {
		id, _ := kernel.ParseID(c.Param("id"))
		var req struct {
			Title    *string   `json:"title"`
			Note     *string   `json:"note"`
			Priority *int      `json:"priority"`
			ListID   *string   `json:"list_id"`  // "" clears
			DueDate  *string   `json:"due_date"` // "" clears
			Tags     *[]string `json:"tags"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tasks.invalid_request", "请求格式错误", err))
			return
		}
		cmd := application.UpdateTaskCmd{
			Owner: owner(c), ID: id, Title: req.Title, Note: req.Note,
			Priority: req.Priority, Tags: req.Tags,
		}
		if req.ListID != nil {
			if *req.ListID == "" {
				cmd.ClearList = true
			} else {
				cmd.ListID = optID(*req.ListID)
			}
		}
		if req.DueDate != nil {
			if *req.DueDate == "" {
				cmd.ClearDue = true
			} else {
				cmd.DueDate = optTime(*req.DueDate)
			}
		}
		t, err := svc.UpdateTask(c.Request.Context(), cmd)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, t)
	})

	r.POST("/tasks/:id/complete", gate(taskRes, "write"), func(c *gin.Context) {
		id, _ := kernel.ParseID(c.Param("id"))
		t, err := svc.SetStatus(c.Request.Context(), owner(c), id, true)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, t)
	})

	r.POST("/tasks/:id/reopen", gate(taskRes, "write"), func(c *gin.Context) {
		id, _ := kernel.ParseID(c.Param("id"))
		t, err := svc.SetStatus(c.Request.Context(), owner(c), id, false)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, t)
	})

	r.DELETE("/tasks/:id", gate(taskRes, "delete"), func(c *gin.Context) {
		id, _ := kernel.ParseID(c.Param("id"))
		if err := svc.DeleteTask(c.Request.Context(), owner(c), id); err != nil {
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
	// Accept RFC3339 or date-only.
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return &t
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return &t
	}
	return nil
}
