// Package books exposes the library-management (图书馆) bounded context as a
// platform Module. Demonstrates the full contract: Manifest (RBAC permissions +
// console menus) + RBAC-gated routes mounted under /api/apps/books/*.
package books

import (
	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/contexts/books/application"
	"github.com/leo/iop/server/internal/contexts/books/infrastructure"
	booksiface "github.com/leo/iop/server/internal/contexts/books/interface"
	"github.com/leo/iop/server/internal/shared/module"
)

type Module struct {
	app *application.Service
}

// New constructs the books module from platform deps. Matches module.Constructor.
func New(deps module.Deps) module.Module {
	repo := infrastructure.NewRepo(deps.Tenant)
	return &Module{app: application.NewService(repo, deps.Bus, deps.Clock)}
}

func (m *Module) Manifest() module.Manifest {
	return module.Manifest{
		Code:        "books",
		Name:        "图书",
		Description: "图书馆管理：图书录入与馆藏管理、按书名/作者/ISBN 检索、在线借阅与归还、借阅记录追踪",
		// Open-book icon.
		Icon:     "M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2zM22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z",
		Color:    "var(--cat-resource)",
		Category: "知识内容",
		Version:  "1.0.0",
		Permissions: []module.Permission{
			{Resource: "books.book", Action: "read", Label: "查看图书 / 借阅记录"},
			{Resource: "books.book", Action: "manage", Label: "管理图书 / 所有借阅记录"},
			{Resource: "books.book", Action: "borrow", Label: "借阅 / 归还图书"},
		},
		Events: []string{
			"books.book_created", "books.borrowed", "books.returned",
		},
		Menus: []module.MenuNode{
			{
				Key: "books.root", Title: "图书", Type: "dir",
				Console: "tenant", App: "books", Order: 60,
				Icon: "M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2zM22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z",
			},
			{
				Key: "books.catalog", Title: "图书借阅", Path: "/books", Parent: "books.root",
				Type: "menu", Console: "tenant", App: "books", Perm: "books.book:read", Order: 61,
				Icon: "M12 7a4 4 0 1 0 0 8 4 4 0 0 0 0-8zM2 21a10 10 0 0 1 20 0",
			},
			{
				Key: "books.manage", Title: "图书管理", Path: "/books/manage", Parent: "books.root",
				Type: "menu", Console: "tenant", App: "books", Perm: "books.book:manage", Order: 62,
				Icon: "M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6zM19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33",
			},
		},
	}
}

// RegisterRoutes mounts /apps/books/... gated by deps.Authz.
func (m *Module) RegisterRoutes(api *gin.RouterGroup, deps module.Deps) {
	booksiface.RegisterRoutes(api, m.app, deps.Authz)
}
