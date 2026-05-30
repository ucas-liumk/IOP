// Package okr exposes the OKR bounded context as a platform Module.
// This file is the canonical example: every business module ships a module.go
// that returns a Manifest + RegisterRoutes implementation.
package okr

import (
	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/contexts/okr/application"
	"github.com/leo/iop/server/internal/contexts/okr/infrastructure"
	okriface "github.com/leo/iop/server/internal/contexts/okr/interface"
	"github.com/leo/iop/server/internal/shared/module"
)

// Module wraps the OKR application service so app.Build can hand off to it.
type Module struct {
	app *application.Service
}

// New constructs an OKR module from platform deps.
// Signature matches module.Constructor.
func New(deps module.Deps) module.Module {
	plans := infrastructure.NewPGPlanRepo(deps.Tenant)
	reports := infrastructure.NewPGReportRepo(deps.Tenant)
	rollup := infrastructure.NewPGRollupQuery(deps.Tenant)
	return &Module{app: application.NewService(plans, reports, rollup, deps.Bus, deps.Clock)}
}

// AppService exposes the underlying service (for legacy /api/plans wiring during transition).
func (m *Module) AppService() *application.Service { return m.app }

// Manifest declares OKR to the platform.
func (m *Module) Manifest() module.Manifest {
	return module.Manifest{
		Code:        "okr",
		Name:        "OKR 工作安排",
		Description: "按年/半年/月/周分层管理目标，配合日报、周报与跨部门汇总",
		// Bullseye icon — matches the left-rail SVG already in use
		Icon:     "M12 22a10 10 0 1 1 0-20 10 10 0 0 1 0 20zM12 16a4 4 0 1 1 0-8 4 4 0 0 1 0 8zm0-3a1 1 0 1 1 0-2 1 1 0 0 1 0 2z",
		Color:    "var(--cat-collab)",
		Category: "协同办公",
		Version:  "1.0.0",
		Permissions: []module.Permission{
			{Resource: "okr.plan", Action: "read", Label: "查看计划"},
			{Resource: "okr.plan", Action: "write", Label: "新建/编辑计划"},
			{Resource: "okr.plan", Action: "delete", Label: "删除计划"},
			{Resource: "okr.report", Action: "read", Label: "查看报告"},
			{Resource: "okr.report", Action: "write", Label: "提交/编辑报告"},
			{Resource: "okr.rollup", Action: "read", Label: "查看团队汇总"},
		},
		Events: []string{
			"okr.plan_created", "okr.plan_item_added", "okr.plan_item_completed",
			"okr.plan_closed",
			"okr.daily_submitted", "okr.weekly_submitted",
		},
	}
}

// RegisterRoutes mounts /apps/okr/... under the caller's group.
// Legacy /api/plans /api/reports /api/rollups stays wired in app.go for now.
func (m *Module) RegisterRoutes(api *gin.RouterGroup, _ module.Deps) {
	okriface.RegisterRoutes(api, m.app)
}
