// Package project exposes the project-management (项目管理 / Teambition / Trello /
// Jira-style Kanban) bounded context as a platform Module. Demonstrates the full
// contract: Manifest (RBAC permissions + events + menus) + RBAC-gated CRUD routes
// over projects / columns / cards, isolated per tenant.
package project

import (
	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/contexts/project/application"
	"github.com/leo/iop/server/internal/contexts/project/infrastructure"
	projectiface "github.com/leo/iop/server/internal/contexts/project/interface"
	"github.com/leo/iop/server/internal/shared/module"
)

type Module struct {
	app *application.Service
}

// New constructs the project module from platform deps. Matches module.Constructor.
func New(deps module.Deps) module.Module {
	repo := infrastructure.NewRepo(deps.Tenant)
	return &Module{app: application.NewService(repo, deps.Bus, deps.Clock)}
}

func (m *Module) Manifest() module.Manifest {
	return module.Manifest{
		Code:        "project",
		Name:        "项目管理",
		Description: "看板式项目管理（仿 Teambition / Trello / Jira）：项目 / 看板列 / 卡片，拖拽流转、负责人、截止日期、优先级",
		// A Kanban / columns glyph.
		Icon:     "M3 3h7v7H3z M14 3h7v11h-7z M14 17h7v4h-7z M3 13h7v8H3z",
		Color:    "var(--cat-collab)",
		Category: "协同办公",
		Version:  "1.0.0",
		Permissions: []module.Permission{
			{Resource: "project", Action: "read", Label: "查看项目"},
			{Resource: "project", Action: "write", Label: "新建/编辑项目与卡片"},
		},
		Events: []string{
			"project.created", "project.card_created", "project.card_moved",
		},
		Menus: []module.MenuNode{
			{
				Key:     "project.home",
				Title:   "项目管理",
				Icon:    "M3 3h7v7H3z M14 3h7v11h-7z M14 17h7v4h-7z M3 13h7v8H3z",
				Path:    "/project",
				Parent:  "",
				Type:    "menu",
				Console: "tenant",
				App:     "project",
				Perm:    "project:read",
				Order:   40,
			},
		},
	}
}

// RegisterRoutes mounts /apps/project/... gated by deps.Authz.
func (m *Module) RegisterRoutes(api *gin.RouterGroup, deps module.Deps) {
	projectiface.RegisterRoutes(api, m.app, deps.Authz)
}
