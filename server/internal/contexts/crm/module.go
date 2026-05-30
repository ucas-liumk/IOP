package crm

import (
	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/contexts/crm/application"
	iface "github.com/leo/iop/server/internal/contexts/crm/interface"
	"github.com/leo/iop/server/internal/shared/module"
)

type Module struct {
	app *application.Service
}

func New(deps module.Deps) module.Module {
	return &Module{
		app: application.NewService(deps.Tenant, deps.Bus, deps.Clock),
	}
}

func (m *Module) Manifest() module.Manifest {
	return module.Manifest{
		Code:        "crm",
		Name:        "客户管理 CRM",
		Description: "由脚手架生成 · 替换为真实描述",
		Icon:        "M12 2 L22 12 L12 22 L2 12 Z",
		Color:       "var(--cat-biz)",
		Category:    "业务管理",
		Version:     "0.1.0",
		Permissions: []module.Permission{
			{Resource: "crm.item", Action: "read",   Label: "查看"},
			{Resource: "crm.item", Action: "write",  Label: "新建/编辑"},
			{Resource: "crm.item", Action: "delete", Label: "删除"},
		},
		Events: []string{
			"crm.item_created",
		},
	}
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, _ module.Deps) {
	iface.RegisterRoutes(api, m.app)
}
