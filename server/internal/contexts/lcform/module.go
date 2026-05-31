// Package lcform exposes the low-code online-form (在线表单 / 简道云·金数据·Airtable-style)
// bounded context as a platform Module. A form definition carries a JSONB field
// schema; members fill it out and the submissions are listed/exported per form.
// Demonstrates the full contract: Manifest (RBAC permissions + events + console
// menus) + RBAC-gated routes mounted under /api/apps/lcform/*.
package lcform

import (
	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/contexts/lcform/application"
	"github.com/leo/iop/server/internal/contexts/lcform/infrastructure"
	lcformiface "github.com/leo/iop/server/internal/contexts/lcform/interface"
	"github.com/leo/iop/server/internal/shared/module"
)

type Module struct {
	app *application.Service
}

// New constructs the lcform module from platform deps. Matches module.Constructor.
func New(deps module.Deps) module.Module {
	repo := infrastructure.NewRepo(deps.Tenant)
	return &Module{app: application.NewService(repo, deps.Bus, deps.Clock)}
}

func (m *Module) Manifest() module.Manifest {
	return module.Manifest{
		Code:        "lcform",
		Name:        "在线表单",
		Description: "低代码在线表单：可视化设计字段（文本/数字/日期/下拉/勾选/金额/电话等），收集数据、分页查看与 CSV 导出，仿简道云 / 金数据 / Airtable",
		// Clipboard-with-lines (form) icon.
		Icon:     "M9 2h6a1 1 0 0 1 1 1v1h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2V3a1 1 0 0 1 1-1z M8 11h8 M8 15h8",
		Color:    "var(--cat-form)",
		Category: "数据工具",
		Version:  "1.0.0",
		Permissions: []module.Permission{
			{Resource: "lcform.def", Action: "manage", Label: "设计 / 管理表单"},
			{Resource: "lcform.entry", Action: "read", Label: "查看表单与数据"},
			{Resource: "lcform.entry", Action: "write", Label: "填写 / 提交表单"},
		},
		Events: []string{
			"lcform.form_created", "lcform.entry_submitted",
		},
		Menus: []module.MenuNode{
			{
				Key: "lcform.root", Title: "在线表单", Type: "dir",
				Console: "tenant", App: "lcform", Order: 70,
				Icon: "M9 2h6a1 1 0 0 1 1 1v1h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2V3a1 1 0 0 1 1-1z M8 11h8 M8 15h8",
			},
			{
				Key: "lcform.center", Title: "表单中心", Path: "/lcform", Parent: "lcform.root",
				Type: "menu", Console: "tenant", App: "lcform", Perm: "lcform.entry:read", Order: 71,
				Icon: "M3 3h7v7H3z M14 3h7v7h-7z M14 14h7v7h-7z M3 14h7v7H3z",
			},
			{
				Key: "lcform.design", Title: "表单设计", Path: "/lcform/design", Parent: "lcform.root",
				Type: "menu", Console: "tenant", App: "lcform", Perm: "lcform.def:manage", Order: 72,
				Icon: "M12 20h9 M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4z",
			},
		},
	}
}

// RegisterRoutes mounts /apps/lcform/... gated by deps.Authz.
func (m *Module) RegisterRoutes(api *gin.RouterGroup, deps module.Deps) {
	lcformiface.RegisterRoutes(api, m.app, deps.Authz)
}
