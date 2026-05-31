// Package docs exposes the knowledge-base (知识库 / 语雀·飞书文档·Notion-style)
// bounded context as a platform Module. A wiki is a tree of folders and docs;
// docs carry rich-text/markdown content. Demonstrates the full module contract:
// Manifest (RBAC permissions + events + menus) + RBAC-gated routes.
package docs

import (
	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/contexts/docs/application"
	"github.com/leo/iop/server/internal/contexts/docs/infrastructure"
	docsiface "github.com/leo/iop/server/internal/contexts/docs/interface"
	"github.com/leo/iop/server/internal/shared/module"
)

type Module struct {
	app *application.Service
}

// New constructs the docs module from platform deps. Matches module.Constructor.
func New(deps module.Deps) module.Module {
	repo := infrastructure.NewRepo(deps.Tenant)
	return &Module{app: application.NewService(repo, deps.Bus, deps.Clock)}
}

func (m *Module) Manifest() module.Manifest {
	return module.Manifest{
		Code:        "docs",
		Name:        "知识库",
		Description: "树形知识库 — 目录 / 文档，富文本编辑与保存，仿语雀 / 飞书文档 / Notion",
		// Book / document icon.
		Icon:     "M4 4a2 2 0 0 1 2-2h9l5 5v13a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V4z M14 2v6h6 M8 13h8 M8 17h8 M8 9h3",
		Color:    "var(--cat-knowledge)",
		Category: "知识内容",
		Version:  "1.0.0",
		Permissions: []module.Permission{
			{Resource: "docs.node", Action: "read", Label: "查看知识库"},
			{Resource: "docs.node", Action: "write", Label: "编辑文档/目录"},
			{Resource: "docs.node", Action: "delete", Label: "删除文档/目录"},
		},
		Events: []string{
			"docs.node_created", "docs.doc_saved",
		},
		Menus: []module.MenuNode{
			{
				Key:     "docs.home",
				Title:   "知识库",
				Icon:    "M4 4a2 2 0 0 1 2-2h9l5 5v13a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V4z M14 2v6h6",
				Path:    "/docs",
				Parent:  "",
				Type:    "menu",
				Console: "tenant",
				App:     "docs",
				Perm:    "docs.node:read",
				Order:   40,
			},
		},
	}
}

// RegisterRoutes mounts /apps/docs/... gated by deps.Authz.
func (m *Module) RegisterRoutes(api *gin.RouterGroup, deps module.Deps) {
	docsiface.RegisterRoutes(api, m.app, deps.Authz)
}
