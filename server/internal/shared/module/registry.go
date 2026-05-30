package module

import (
	"sort"
	"sync"

	"github.com/gin-gonic/gin"
)

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

// MountAll wires every module's routes under /api/apps/<code>/.
// Caller passes an *already-authenticated* gin.RouterGroup so each module's
// routes inherit the platform auth + tenant-loader middleware chain.
func (r *Registry) MountAll(api *gin.RouterGroup, deps Deps) {
	for _, m := range r.All() {
		manifest := m.Manifest()
		group := api.Group("/apps/" + manifest.Code)
		m.RegisterRoutes(group, deps)
	}
}
