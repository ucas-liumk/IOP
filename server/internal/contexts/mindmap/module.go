// Package mindmap exposes the mind-map (思维导图 / ProcessOn / GitMind / 百度脑图-style)
// bounded context as a platform Module. Demonstrates the full contract:
// Manifest (RBAC permissions + events + menus) + RBAC-gated CRUD routes over a
// JSONB node-tree document, isolated per tenant.
package mindmap

import (
	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/contexts/mindmap/application"
	"github.com/leo/iop/server/internal/contexts/mindmap/infrastructure"
	mindmapiface "github.com/leo/iop/server/internal/contexts/mindmap/interface"
	"github.com/leo/iop/server/internal/shared/module"
)

type Module struct {
	app *application.Service
}

// New constructs the mindmap module from platform deps. Matches module.Constructor.
func New(deps module.Deps) module.Module {
	repo := infrastructure.NewRepo(deps.Tenant)
	return &Module{app: application.NewService(repo, deps.Bus, deps.Clock)}
}

func (m *Module) Manifest() module.Manifest {
	return module.Manifest{
		Code:        "mindmap",
		Name:        "思维导图",
		Description: "在线思维导图 / 脑图编辑器（仿 ProcessOn / GitMind / 百度脑图）：节点树、画布编辑、保存为 JSON 文档",
		// A branching-tree / share-node glyph.
		Icon:     "M18 16.08c-.76 0-1.44.3-1.96.77L8.91 12.7c.05-.23.09-.46.09-.7s-.04-.47-.09-.7l7.05-4.11c.54.5 1.25.81 2.04.81 1.66 0 3-1.34 3-3s-1.34-3-3-3-3 1.34-3 3c0 .24.04.47.09.7L8.04 9.81C7.5 9.31 6.79 9 6 9c-1.66 0-3 1.34-3 3s1.34 3 3 3c.79 0 1.5-.31 2.04-.81l7.12 4.16c-.05.21-.08.43-.08.65 0 1.61 1.31 2.92 2.92 2.92s2.92-1.31 2.92-2.92-1.31-2.92-2.92-2.92z",
		Color:    "var(--cat-creative)",
		Category: "知识内容",
		Version:  "1.0.0",
		Permissions: []module.Permission{
			{Resource: "mindmap.map", Action: "read", Label: "查看思维导图"},
			{Resource: "mindmap.map", Action: "write", Label: "新建/编辑思维导图"},
			{Resource: "mindmap.map", Action: "delete", Label: "删除思维导图"},
		},
		Events: []string{
			"mindmap.created", "mindmap.updated",
		},
		Menus: []module.MenuNode{
			{
				Key:     "mindmap.home",
				Title:   "思维导图",
				Icon:    "M18 16.08c-.76 0-1.44.3-1.96.77L8.91 12.7c.05-.23.09-.46.09-.7s-.04-.47-.09-.7l7.05-4.11c.54.5 1.25.81 2.04.81 1.66 0 3-1.34 3-3s-1.34-3-3-3-3 1.34-3 3c0 .24.04.47.09.7L8.04 9.81C7.5 9.31 6.79 9 6 9c-1.66 0-3 1.34-3 3s1.34 3 3 3c.79 0 1.5-.31 2.04-.81l7.12 4.16c-.05.21-.08.43-.08.65 0 1.61 1.31 2.92 2.92 2.92s2.92-1.31 2.92-2.92-1.31-2.92-2.92-2.92z",
				Path:    "/mindmap",
				Parent:  "",
				Type:    "menu",
				Console: "tenant",
				App:     "mindmap",
				Perm:    "mindmap.map:read",
				Order:   30,
			},
		},
	}
}

// RegisterRoutes mounts /apps/mindmap/... gated by deps.Authz.
func (m *Module) RegisterRoutes(api *gin.RouterGroup, deps module.Deps) {
	mindmapiface.RegisterRoutes(api, m.app, deps.Authz)
}
