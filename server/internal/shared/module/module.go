// Package module defines the contract every business module implements.
// To add a new module: implement Module interface + register with app.Registry.
// See contexts/okr for the canonical example.
package module

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/leo/iop/server/internal/shared/eventbus"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/tenantdb"
)

// Module is the public contract for a business module.
//
// Lifecycle:
//   1. app.Build() creates each module via its constructor (gets Deps)
//   2. app.Registry.Register(mod) collects them
//   3. Registry.MountAll() calls Manifest() + RegisterRoutes() on each
//   4. Registry.AllPermissions() collects declared RBAC resources/actions
//   5. AppCenter / catalog API surfaces manifests to the UI
type Module interface {
	Manifest() Manifest
	RegisterRoutes(api *gin.RouterGroup, deps Deps)
}

// Manifest describes a module's identity, UI, and permission contract.
// All fields except Migrations are also surfaced to the frontend via /apps/catalog.
type Manifest struct {
	Code        string       `json:"code"`        // "okr" — unique, used for route prefix + DB
	Name        string       `json:"name"`        // "OKR 工作安排"
	Description string       `json:"description"` // shown in AppCenter
	Icon        string       `json:"icon"`        // SVG path data
	Color       string       `json:"color"`       // CSS color (var(--cat-collab))
	Category    string       `json:"category"`    // "协同办公" | "业务管理" | ...
	Version     string       `json:"version"`     // "1.0.0"
	Permissions []Permission `json:"permissions"` // RBAC resource×action declarations
	Events      []string     `json:"events"`      // event topics this module publishes
}

// Permission is one (resource, action) tuple a module exposes to RBAC.
// Used by the role-editor UI to render dropdowns instead of free text inputs.
type Permission struct {
	Resource string `json:"resource"` // "okr.plan"
	Action   string `json:"action"`   // "read" | "write" | "delete" | "*"
	Label    string `json:"label"`    // 中文展示名："查看计划"
}

// Deps is what app.Build() injects into each module constructor.
// Modules NEVER reach for globals — everything they need is here.
type Deps struct {
	Pool     *pgxpool.Pool
	Tenant   *tenantdb.TenantDB
	Platform *tenantdb.PlatformDB
	Bus      eventbus.Bus
	Logger   *zap.Logger
	Clock    kernel.Clock
}

// Constructor is the function signature every module's New() must satisfy.
// Stored in the Registry so app boot can call them in dependency order.
type Constructor func(deps Deps) Module

// Hook gives modules a chance to do tenant-scoped setup (e.g. seed dictionary entries)
// when a new tenant is provisioned. Optional — modules that don't implement it are skipped.
type Hook interface {
	OnTenantProvision(ctx context.Context, tenantID kernel.ID) error
}
