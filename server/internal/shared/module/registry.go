package module

import (
	"sort"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
)

// appEnabledGate returns 403 unless the app is enabled for the caller: either
// the caller's tenant has installed it (org-level) OR the caller added it to
// their own per-user workspace.
func appEnabledGate(check AppEnabledFunc, code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		tid, ok := kernel.TenantIDFromContext(ctx)
		if !ok || tid == "" {
			apiresp.Fail(c, errors.New(errors.KindForbidden, "app.no_tenant", "请先选择租户"))
			return
		}
		puid, _ := kernel.PlatformUserIDFromContext(ctx)
		enabled, err := check(ctx, tid, puid, code)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		if !enabled {
			apiresp.Fail(c, errors.New(errors.KindForbidden, "app.not_enabled", "应用未在本租户启用"))
			return
		}
		c.Next()
	}
}

// Registry collects modules at boot, mounts their routes, and exposes manifests.
// Goroutine-safe so cmd/tenantctl and cmd/server can both build a registry.
type Registry struct {
	mu      sync.RWMutex
	modules map[string]Module // keyed by Manifest().Code
}

func NewRegistry() *Registry {
	return &Registry{modules: make(map[string]Module)}
}

// Register adds a module. Later registrations for the same Code override earlier.
func (r *Registry) Register(m Module) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.modules[m.Manifest().Code] = m
}

// Get returns the module with the given code, or nil.
func (r *Registry) Get(code string) Module {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.modules[code]
}

// All returns every registered module, sorted by Code for stable iteration.
func (r *Registry) All() []Module {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Module, 0, len(r.modules))
	for _, m := range r.modules {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Manifest().Code < out[j].Manifest().Code })
	return out
}

// Manifests returns the catalog: every module's metadata. Used by /apps/catalog.
func (r *Registry) Manifests() []Manifest {
	mods := r.All()
	out := make([]Manifest, 0, len(mods))
	for _, m := range mods {
		out = append(out, m.Manifest())
	}
	return out
}

// AllPermissions aggregates every Permission declared by every module.
// Returns a flat list (modules' permissions are namespaced by Resource).
// Used by /admin/permissions to populate role-editor dropdowns.
func (r *Registry) AllPermissions() []Permission {
	mods := r.All()
	out := []Permission{}
	for _, m := range mods {
		out = append(out, m.Manifest().Permissions...)
	}
	return out
}

// AllMenus aggregates every MenuNode declared by every module.
// Returns a flat list (built-in console menus are added separately by the app
// layer). Used to assemble the console menu catalog / nav trees.
func (r *Registry) AllMenus() []MenuNode {
	mods := r.All()
	out := []MenuNode{}
	for _, m := range mods {
		out = append(out, m.Manifest().Menus...)
	}
	return out
}

// MountAll wires every module's routes under /api/apps/<code>/.
// Caller passes an *already-authenticated* gin.RouterGroup so each module's
// routes inherit the platform auth + tenant-loader middleware chain.
//
// When deps.AppEnabled is set, each module group is gated by an enablement check:
// if the caller's tenant has not enabled the app (AppStore), requests get 403.
// This makes "disable app" a real access boundary, not just a UI toggle.
func (r *Registry) MountAll(api *gin.RouterGroup, deps Deps) {
	for _, m := range r.All() {
		code := m.Manifest().Code
		group := api.Group("/apps/" + code)
		if deps.AppEnabled != nil {
			group.Use(appEnabledGate(deps.AppEnabled, code))
		}
		m.RegisterRoutes(group, deps)
	}
}
