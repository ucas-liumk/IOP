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
//  1. app.Build() creates each module via its constructor (gets Deps)
//  2. app.Registry.Register(mod) collects them
//  3. Registry.MountAll() calls Manifest() + RegisterRoutes() on each
//  4. Registry.AllPermissions() collects declared RBAC resources/actions
//  5. AppCenter / catalog API surfaces manifests to the UI
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
	Menus       []MenuNode   `json:"menus"`       // console nav nodes this module contributes
}

// MenuNode is one node in a console's navigation/permission catalog. Modules
// declare them in their Manifest; the platform aggregates them (plus built-in
// console menus) into a tree. A node is VISIBLE to a user when: Console matches
// the current console; AND (Perm == "" OR the user's role policies permit the
// split "resource:action"); AND (App == "" OR that app is enabled for the tenant).
type MenuNode struct {
	Key     string `json:"key"`     // unique, e.g. "okr.plans"
	Title   string `json:"title"`   // "我的计划"
	Icon    string `json:"icon"`    // SVG path data
	Path    string `json:"path"`    // frontend route, e.g. "/okr/plans" (dir nodes may be empty)
	Parent  string `json:"parent"`  // parent node Key; "" = top level
	Type    string `json:"type"`    // "dir" | "menu" | "button"
	Console string `json:"console"` // "platform" | "tenant" | "both"
	App     string `json:"app"`     // owning module code (gated by AppEnabled); "" for built-in
	Perm    string `json:"perm"`    // required permission "resource:action"; "" = login/App only
	Order   int    `json:"order"`
}

// Permission is one (resource, action) tuple a module exposes to RBAC.
// Used by the role-editor UI to render dropdowns instead of free text inputs.
type Permission struct {
	Resource string `json:"resource"` // "okr.plan"
	Action   string `json:"action"`   // "read" | "write" | "delete" | "*"
	Label    string `json:"label"`    // 中文展示名："查看计划"
}

// AuthzFunc returns a gin middleware that enforces a declared RBAC permission
// (resource×action) on a route. Modules use it to gate their own endpoints
// WITHOUT importing the iam package directly — the platform wires it from
// iam.RBAC at boot. The resource/action must match a Manifest.Permission so the
// role editor can grant it. Admins (tenant_admin / platform_admin) bypass.
type AuthzFunc func(resource, action string) gin.HandlerFunc

// AppEnabledFunc reports whether a tenant has enabled (installed) the given app
// code via the AppStore. Used by the Registry to gate module routes so that
// disabling an app actually blocks its API, not just its UI visibility.
type AppEnabledFunc func(ctx context.Context, tenantID kernel.ID, code string) (bool, error)

// Deps is what app.Build() injects into each module constructor.
// Modules NEVER reach for globals — everything they need is here.
type Deps struct {
	Pool     *pgxpool.Pool
	Tenant   *tenantdb.TenantDB
	Platform *tenantdb.PlatformDB
	Bus      eventbus.Bus
	Logger   *zap.Logger
	Clock    kernel.Clock
	// Authz gates module routes by declared permission. Never nil in production
	// wiring; modules should still guard against nil for unit tests.
	Authz AuthzFunc
	// AppEnabled, when set, gates each module's routes on tenant AppStore enablement.
	AppEnabled AppEnabledFunc
}

// Constructor is the function signature every module's New() must satisfy.
// Stored in the Registry so app boot can call them in dependency order.
type Constructor func(deps Deps) Module

// Hook gives modules a chance to do tenant-scoped setup (e.g. seed dictionary entries)
// when a new tenant is provisioned. Optional — modules that don't implement it are skipped.
type Hook interface {
	OnTenantProvision(ctx context.Context, tenantID kernel.ID) error
}

// InstallHook lets a module run tenant-scoped setup when an admin ENABLES it for
// a tenant via the AppStore (e.g. seed default data/config). Optional — modules
// that don't implement it are skipped. If OnInstall returns an error the AppStore
// rolls back the enablement so install is all-or-nothing.
type InstallHook interface {
	OnInstall(ctx context.Context, tenantID kernel.ID) error
}

// UninstallHook lets a module clean up (or warn about) tenant data when an admin
// DISABLES it. Optional.
type UninstallHook interface {
	OnUninstall(ctx context.Context, tenantID kernel.ID) error
}
