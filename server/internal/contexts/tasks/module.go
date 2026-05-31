// Package tasks exposes the task-management (滴答清单-style) bounded context as a
// platform Module. Second reference module after OKR — demonstrates the full
// contract: Manifest (with RBAC permissions + events) + RBAC-gated routes.
package tasks

import (
	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/contexts/tasks/application"
	"github.com/leo/iop/server/internal/contexts/tasks/infrastructure"
	tasksiface "github.com/leo/iop/server/internal/contexts/tasks/interface"
	"github.com/leo/iop/server/internal/shared/module"
)

type Module struct {
	app *application.Service
}

// New constructs the tasks module from platform deps. Matches module.Constructor.
func New(deps module.Deps) module.Module {
	repo := infrastructure.NewRepo(deps.Tenant)
	return &Module{app: application.NewService(repo, deps.Bus, deps.Clock)}
}

func (m *Module) Manifest() module.Manifest {
	return module.Manifest{
		Code:        "tasks",
		Name:        "任务清单",
		Description: "清单 / 任务 / 子任务、优先级、截止日期、标签与「今天 / 最近7天 / 已完成」智能视图",
		// Checklist icon.
		Icon:     "M9 16.17 4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z M3 5h12v2H3z M3 11h8v2H3z",
		Color:    "var(--cat-task)",
		Category: "工作协同",
		Version:  "1.0.0",
		Permissions: []module.Permission{
			{Resource: "tasks.list", Action: "read", Label: "查看清单"},
			{Resource: "tasks.list", Action: "write", Label: "新建/编辑清单"},
			{Resource: "tasks.list", Action: "delete", Label: "删除清单"},
			{Resource: "tasks.task", Action: "read", Label: "查看任务"},
			{Resource: "tasks.task", Action: "write", Label: "新建/编辑任务"},
			{Resource: "tasks.task", Action: "delete", Label: "删除任务"},
		},
		Events: []string{
			"tasks.list_created", "tasks.task_created",
			"tasks.task_completed", "tasks.task_reopened",
		},
		Menus: []module.MenuNode{
			{Key: "tasks.home", Title: "任务清单", Path: "/tasks",
				Type: "menu", Console: "tenant", App: "tasks", Perm: "tasks.task:read", Order: 40,
				Icon: "M9 16.17 4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z M3 5h12v2H3z M3 11h8v2H3z"},
		},
	}
}

// RegisterRoutes mounts /apps/tasks/... gated by deps.Authz.
func (m *Module) RegisterRoutes(api *gin.RouterGroup, deps module.Deps) {
	tasksiface.RegisterRoutes(api, m.app, deps.Authz)
}
