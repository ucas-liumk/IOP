// Package approval exposes the approval-center (审批中心, 钉钉审批/飞书审批-style)
// bounded context as a platform Module. Demonstrates the full contract: Manifest
// (RBAC permissions + console menus + events) + RBAC-gated routes mounted under
// /api/apps/approval/* by the Registry.
package approval

import (
	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/contexts/approval/application"
	"github.com/leo/iop/server/internal/contexts/approval/infrastructure"
	approvaliface "github.com/leo/iop/server/internal/contexts/approval/interface"
	"github.com/leo/iop/server/internal/shared/module"
)

type Module struct {
	app *application.Service
}

// New constructs the approval module from platform deps. Matches module.Constructor.
func New(deps module.Deps) module.Module {
	repo := infrastructure.NewRepo(deps.Tenant)
	return &Module{app: application.NewService(repo, deps.Bus, deps.Clock)}
}

func (m *Module) Manifest() module.Manifest {
	const app = "approval"
	return module.Manifest{
		Code:        app,
		Name:        "审批中心",
		Description: "自定义审批模板（动态表单 + 审批流）、发起审批、多级审批/抄送、待办/已办/我发起/抄送我，仿钉钉审批·飞书审批",
		// Document-with-check icon.
		Icon:     "M9 11l3 3L22 4 M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11",
		Color:    "var(--cat-collab)",
		Category: "协同办公",
		Version:  "1.0.0",
		Permissions: []module.Permission{
			{Resource: "approval.form", Action: "manage", Label: "管理审批模板"},
			{Resource: "approval.instance", Action: "submit", Label: "发起审批"},
			{Resource: "approval.instance", Action: "approve", Label: "审批处理"},
			{Resource: "approval.instance", Action: "read", Label: "查看审批"},
		},
		Events: []string{
			"approval.form_created",
			"approval.instance_submitted",
			"approval.instance_finished",
		},
		Menus: []module.MenuNode{
			{
				Key: "approval.root", Title: "审批中心", Type: "dir",
				Console: "tenant", App: app, Order: 60,
				Icon: "M9 11l3 3L22 4 M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11",
			},
			{
				Key: "approval.mine", Title: "我的审批", Path: "/approval/mine", Parent: "approval.root",
				Type: "menu", Console: "tenant", App: app, Perm: "approval.instance:read", Order: 61,
				Icon: "M9 11l3 3L22 4 M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11",
			},
			{
				Key: "approval.new", Title: "发起审批", Path: "/approval/new", Parent: "approval.root",
				Type: "menu", Console: "tenant", App: app, Perm: "approval.instance:submit", Order: 62,
				Icon: "M12 5v14M5 12h14",
			},
			{
				Key: "approval.forms", Title: "模板管理", Path: "/approval/forms", Parent: "approval.root",
				Type: "menu", Console: "tenant", App: app, Perm: "approval.form:manage", Order: 63,
				Icon: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8zM14 2v6h6M9 13h6M9 17h6",
			},
		},
	}
}

// RegisterRoutes mounts /apps/approval/... gated by deps.Authz.
func (m *Module) RegisterRoutes(api *gin.RouterGroup, deps module.Deps) {
	approvaliface.RegisterRoutes(api, m.app, deps.Authz)
}
