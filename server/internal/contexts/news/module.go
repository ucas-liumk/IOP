// Package news exposes the 时政资讯 (gov news CMS, modeled on 政务门户 / 人民网频道)
// bounded context as a platform Module. Demonstrates the full contract: Manifest
// (RBAC permissions + events + console menus) + RBAC-gated routes.
package news

import (
	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/contexts/news/application"
	"github.com/leo/iop/server/internal/contexts/news/infrastructure"
	newsiface "github.com/leo/iop/server/internal/contexts/news/interface"
	"github.com/leo/iop/server/internal/shared/module"
)

type Module struct {
	app *application.Service
}

// New constructs the news module from platform deps. Matches module.Constructor.
func New(deps module.Deps) module.Module {
	repo := infrastructure.NewRepo(deps.Tenant)
	return &Module{app: application.NewService(repo, deps.Bus, deps.Clock)}
}

func (m *Module) Manifest() module.Manifest {
	return module.Manifest{
		Code:        "news",
		Name:        "时政资讯",
		Description: "政务门户式资讯频道：栏目分类、文章撰写 / 发布 / 下线，面向员工的资讯阅读与详情。",
		// Newspaper / article icon.
		Icon:     "M4 4h16v16H4zM6 7h7v2H6zm0 4h12v2H6zm0 4h12v2H6zM15 7h3v2h-3z",
		Color:    "var(--cat-collab)",
		Category: "协同办公",
		Version:  "1.0.0",
		Permissions: []module.Permission{
			{Resource: "news.read", Action: "read", Label: "阅读资讯"},
			{Resource: "news.manage", Action: "read", Label: "查看后台文章"},
			{Resource: "news.manage", Action: "write", Label: "撰写/编辑/发布资讯"},
			{Resource: "news.manage", Action: "delete", Label: "删除栏目/文章"},
		},
		Events: []string{
			"news.article_created", "news.article_published", "news.article_unpublished",
		},
		Menus: []module.MenuNode{
			{
				Key: "news.dir", Title: "时政资讯",
				Icon:    "M4 4h16v16H4zM6 7h7v2H6zm0 4h12v2H6zm0 4h12v2H6zM15 7h3v2h-3z",
				Type:    "dir",
				Console: "tenant", App: "news", Order: 50,
			},
			{
				Key: "news.feed", Title: "资讯阅读", Path: "/news",
				Icon:   "M4 4h16v16H4zM6 7h7v2H6zm0 4h12v2H6zm0 4h12v2H6zM15 7h3v2h-3z",
				Parent: "news.dir", Type: "menu",
				Console: "tenant", App: "news", Perm: "news.read:read", Order: 1,
			},
			{
				Key: "news.manage", Title: "内容管理", Path: "/news/manage",
				Icon:   "M3 17.25V21h3.75L17.81 9.94l-3.75-3.75zM20.71 7.04a1 1 0 0 0 0-1.41l-2.34-2.34a1 1 0 0 0-1.41 0l-1.83 1.83 3.75 3.75z",
				Parent: "news.dir", Type: "menu",
				Console: "tenant", App: "news", Perm: "news.manage:read", Order: 2,
			},
		},
	}
}

// RegisterRoutes mounts /apps/news/... gated by deps.Authz.
func (m *Module) RegisterRoutes(api *gin.RouterGroup, deps module.Deps) {
	newsiface.RegisterRoutes(api, m.app, deps.Authz)
}
