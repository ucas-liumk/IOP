// Package iface wires the approval module's REST routes. Mounted under
// /api/apps/approval/* by the module Registry; every route is RBAC-gated via the
// module.AuthzFunc passed in (resource×action from the Manifest).
package iface

import (
	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/contexts/approval/application"
	"github.com/leo/iop/server/internal/contexts/approval/domain"
	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/services/iam"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/module"
)

const (
	resForm     = "approval.form"     // action: manage
	resInstance = "approval.instance" // actions: submit / approve / read
)

func RegisterRoutes(r *gin.RouterGroup, svc *application.Service, authz module.AuthzFunc) {
	gate := func(resource, action string) gin.HandlerFunc {
		if authz == nil {
			return func(c *gin.Context) { c.Next() }
		}
		return authz(resource, action)
	}
	claims := func(c *gin.Context) *iam.Claims {
		cl, _ := iam.ClaimsFromContext(c.Request.Context())
		return cl
	}
	member := func(c *gin.Context) kernel.ID {
		if cl := claims(c); cl != nil {
			return cl.MemberID
		}
		return ""
	}
	tenant := func(c *gin.Context) kernel.ID {
		if cl := claims(c); cl != nil {
			return cl.TenantID
		}
		return ""
	}

	// ---- Directory (for assignee pickers) ----
	r.GET("/members", gate(resInstance, "read"), func(c *gin.Context) {
		ms, err := svc.ListMembers(c.Request.Context())
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"members": ms})
	})

	// ---- Forms (templates) ----
	r.GET("/forms", gate(resInstance, "read"), func(c *gin.Context) {
		includeDisabled := c.Query("all") == "1"
		forms, err := svc.ListForms(c.Request.Context(), includeDisabled)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"forms": forms})
	})

	r.GET("/forms/:id", gate(resInstance, "read"), func(c *gin.Context) {
		id, _ := kernel.ParseID(c.Param("id"))
		f, err := svc.GetForm(c.Request.Context(), id)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, f)
	})

	r.POST("/forms", gate(resForm, "manage"), func(c *gin.Context) {
		req, err := bindForm(c)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		f, err := svc.CreateForm(c.Request.Context(), application.SaveFormCmd{
			Code: req.Code, Name: req.Name, Icon: req.Icon, Description: req.Description,
			Fields: req.Fields, Flow: req.Flow, Status: req.Status, Actor: member(c),
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.Created(c, f)
	})

	r.PUT("/forms/:id", gate(resForm, "manage"), func(c *gin.Context) {
		id, _ := kernel.ParseID(c.Param("id"))
		req, err := bindForm(c)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		f, err := svc.UpdateForm(c.Request.Context(), application.SaveFormCmd{
			ID: id, Code: req.Code, Name: req.Name, Icon: req.Icon, Description: req.Description,
			Fields: req.Fields, Flow: req.Flow, Status: req.Status, Actor: member(c),
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, f)
	})

	r.DELETE("/forms/:id", gate(resForm, "manage"), func(c *gin.Context) {
		id, _ := kernel.ParseID(c.Param("id"))
		if err := svc.DeleteForm(c.Request.Context(), id); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	// ---- Instances ----
	r.POST("/instances", gate(resInstance, "submit"), func(c *gin.Context) {
		var req struct {
			FormID string         `json:"form_id" binding:"required"`
			Data   map[string]any `json:"data"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "approval.invalid_request", "请求格式错误", err))
			return
		}
		formID, err := kernel.ParseID(req.FormID)
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "approval.invalid_id", "模板 id 无效", err))
			return
		}
		if req.Data == nil {
			req.Data = map[string]any{}
		}
		ins, err := svc.Submit(c.Request.Context(), application.SubmitCmd{
			TenantID: tenant(c), FormID: formID, Data: req.Data, Initiator: member(c),
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.Created(c, ins)
	})

	r.GET("/instances", gate(resInstance, "read"), func(c *gin.Context) {
		typ := c.Query("type")
		if typ == "" {
			typ = "todo"
		}
		data, err := svc.Inbox(c.Request.Context(), domain.TaskQuery{Member: member(c), Type: typ})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"type": typ, "items": data})
	})

	r.GET("/instances/:id", gate(resInstance, "read"), func(c *gin.Context) {
		id, _ := kernel.ParseID(c.Param("id"))
		ins, err := svc.GetInstance(c.Request.Context(), id, member(c))
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, ins)
	})

	r.POST("/instances/:id/cancel", gate(resInstance, "submit"), func(c *gin.Context) {
		id, _ := kernel.ParseID(c.Param("id"))
		if err := svc.CancelInstance(c.Request.Context(), id, member(c)); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	// ---- Tasks ----
	r.POST("/tasks/:id/act", gate(resInstance, "approve"), func(c *gin.Context) {
		id, _ := kernel.ParseID(c.Param("id"))
		var req struct {
			Action  string `json:"action" binding:"required"` // approve / reject / read
			Comment string `json:"comment"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "approval.invalid_request", "请求格式错误", err))
			return
		}
		if req.Action != "approve" && req.Action != "reject" && req.Action != "read" {
			apiresp.Fail(c, errors.New(errors.KindParam, "approval.bad_action", "操作无效"))
			return
		}
		if err := svc.Act(c.Request.Context(), application.ActCmd{
			TenantID: tenant(c), TaskID: id, Member: member(c),
			Approve: req.Action == "approve", Comment: req.Comment,
		}); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})
}

type formReq struct {
	Code        string            `json:"code"`
	Name        string            `json:"name" binding:"required"`
	Icon        string            `json:"icon"`
	Description string            `json:"description"`
	Fields      []domain.Field    `json:"fields"`
	Flow        []domain.FlowNode `json:"flow"`
	Status      string            `json:"status"`
}

func bindForm(c *gin.Context) (*formReq, error) {
	var req formReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return nil, errors.Wrap(errors.KindParam, "approval.invalid_request", "请求格式错误", err)
	}
	return &req, nil
}
