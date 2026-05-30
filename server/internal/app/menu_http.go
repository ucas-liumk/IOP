package app

import (
	"context"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/services/iam"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/module"
)

// menuNodeView is the JSON shape returned by the menu APIs: a MenuNode plus its
// nested children (assembled by Parent).
type menuNodeView struct {
	module.MenuNode
	Children []*menuNodeView `json:"children"`
}

// allMenus returns the full console catalog = built-in console menus + every
// module-declared menu, in a stable order.
func (a *App) allMenus() []module.MenuNode {
	out := append([]module.MenuNode{}, builtinMenus()...)
	out = append(out, a.Modules.AllMenus()...)
	return out
}

// consoleMatches reports whether a node belongs to the requested console
// ("tenant"|"platform"). "both" matches either.
func consoleMatches(node module.MenuNode, console string) bool {
	return node.Console == console || node.Console == "both"
}

// buildMenuTree turns a flat node list into a nested tree by Parent. Nodes are
// sorted by (Order, Key). Orphans (Parent not present in the set, e.g. because a
// parent dir was filtered out) are promoted to top level so they stay reachable.
func buildMenuTree(nodes []module.MenuNode) []*menuNodeView {
	byKey := make(map[string]*menuNodeView, len(nodes))
	for _, n := range nodes {
		byKey[n.Key] = &menuNodeView{MenuNode: n, Children: []*menuNodeView{}}
	}
	roots := []*menuNodeView{}
	for _, n := range nodes {
		v := byKey[n.Key]
		if n.Parent != "" {
			if parent, ok := byKey[n.Parent]; ok {
				parent.Children = append(parent.Children, v)
				continue
			}
		}
		roots = append(roots, v)
	}
	var sortRec func(list []*menuNodeView)
	sortRec = func(list []*menuNodeView) {
		sort.SliceStable(list, func(i, j int) bool {
			if list[i].Order != list[j].Order {
				return list[i].Order < list[j].Order
			}
			return list[i].Key < list[j].Key
		})
		for _, v := range list {
			sortRec(v.Children)
		}
	}
	sortRec(roots)
	return roots
}

// splitPerm splits a "resource:action" perm string. Returns ("","") when empty.
func splitPerm(perm string) (resource, action string) {
	if perm == "" {
		return "", ""
	}
	if i := strings.LastIndex(perm, ":"); i >= 0 {
		return perm[:i], perm[i+1:]
	}
	return perm, "*"
}

// visibleMenus filters nodes for a console by the user's effective rules and
// (for the tenant console) app enablement. A node is kept when:
//   - it belongs to the console, AND
//   - Perm == "" OR the rules permit the split resource:action, AND
//   - App == "" OR (platform console — App gating skipped) OR the app is enabled
//     for the tenant.
//
// Dir nodes whose every child was filtered out are dropped (no empty dirs).
func (a *App) visibleMenus(ctx context.Context, console string, rules []iam.PolicyRule, tenantID kernel.ID) []module.MenuNode {
	enabledCache := map[string]bool{}
	appEnabled := func(code string) bool {
		if console == "platform" {
			return true // platform console is not app-gated
		}
		if tenantID == "" {
			return false
		}
		if v, ok := enabledCache[code]; ok {
			return v
		}
		on, err := a.AppStore.IsInstalled(ctx, tenantID, code)
		if err != nil {
			on = false
		}
		enabledCache[code] = on
		return on
	}

	kept := []module.MenuNode{}
	for _, n := range a.allMenus() {
		if !consoleMatches(n, console) {
			continue
		}
		if n.Perm != "" {
			res, act := splitPerm(n.Perm)
			if !iam.PermitsRule(rules, res, act) {
				continue
			}
		}
		if n.App != "" && !appEnabled(n.App) {
			continue
		}
		kept = append(kept, n)
	}
	return pruneEmptyDirs(kept)
}

// pruneEmptyDirs removes dir nodes that have no surviving descendants. A leaf
// (menu/button) is always kept; a dir is kept only if at least one of its
// children survives.
func pruneEmptyDirs(nodes []module.MenuNode) []module.MenuNode {
	present := make(map[string]module.MenuNode, len(nodes))
	childrenOf := map[string][]string{}
	for _, n := range nodes {
		present[n.Key] = n
		if n.Parent != "" {
			childrenOf[n.Parent] = append(childrenOf[n.Parent], n.Key)
		}
	}
	var hasLiveDescendant func(key string) bool
	hasLiveDescendant = func(key string) bool {
		for _, ck := range childrenOf[key] {
			c := present[ck]
			if c.Type != "dir" || hasLiveDescendant(ck) {
				return true
			}
		}
		return false
	}
	out := []module.MenuNode{}
	for _, n := range nodes {
		if n.Type == "dir" && !hasLiveDescendant(n.Key) {
			continue
		}
		out = append(out, n)
	}
	return out
}

// effectivePerms returns the flat list of "resource:action" perm strings the
// user's rules grant (incl. "*:*"). Used by the frontend for button-level gating
// with a wildcard-aware client check.
func effectivePerms(rules []iam.PolicyRule) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, r := range rules {
		if r.Effect != "allow" {
			continue
		}
		key := r.Resource + ":" + r.Action
		if !seen[key] {
			seen[key] = true
			out = append(out, key)
		}
	}
	sort.Strings(out)
	return out
}

// rulesFor resolves the caller's effective rules for the given console from their
// claims. For the tenant console it uses member/tenant rules (empty when there is
// no tenant context — e.g. a platform-only user). For the platform console it uses
// platform-role policies.
func (a *App) rulesFor(ctx context.Context, console string, claims *iam.Claims) ([]iam.PolicyRule, error) {
	if console == "platform" {
		return a.IAM.PlatformPolicies(ctx, claims.PlatformUserID)
	}
	if claims.MemberID == "" || claims.TenantID == "" {
		return []iam.PolicyRule{}, nil
	}
	return a.IAM.MemberPerms(ctx, claims.MemberID, claims.TenantID)
}

// normalizeConsole maps the ?console= query to a known console, defaulting to tenant.
func normalizeConsole(raw string) string {
	if raw == "platform" {
		return "platform"
	}
	return "tenant"
}

// RegisterMeMenuRoutes mounts /me/menus + /me/perms (login-only; tenant console
// reads tenant context from claims when present). Wire on the authT group.
func (a *App) RegisterMeMenuRoutes(r *gin.RouterGroup) {
	r.GET("/me/menus", func(c *gin.Context) {
		claims, ok := iam.ClaimsFromContext(c.Request.Context())
		if !ok {
			apiresp.OK(c, gin.H{"menus": []*menuNodeView{}})
			return
		}
		console := normalizeConsole(c.Query("console"))
		ctx := c.Request.Context()
		rules, err := a.rulesFor(ctx, console, claims)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		visible := a.visibleMenus(ctx, console, rules, claims.TenantID)
		apiresp.OK(c, gin.H{"menus": buildMenuTree(visible)})
	})

	r.GET("/me/perms", func(c *gin.Context) {
		claims, ok := iam.ClaimsFromContext(c.Request.Context())
		if !ok {
			apiresp.OK(c, gin.H{"perms": []string{}})
			return
		}
		console := normalizeConsole(c.Query("console"))
		ctx := c.Request.Context()
		rules, err := a.rulesFor(ctx, console, claims)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"perms": effectivePerms(rules)})
	})
}

// RegisterTenantMenuCatalogRoute mounts GET /admin/menus — the COMPLETE tenant
// console tree (unfiltered) for the role editor. Wire on the admin group.
func (a *App) RegisterTenantMenuCatalogRoute(r *gin.RouterGroup) {
	r.GET("/admin/menus", func(c *gin.Context) {
		nodes := []module.MenuNode{}
		for _, n := range a.allMenus() {
			if consoleMatches(n, "tenant") {
				nodes = append(nodes, n)
			}
		}
		apiresp.OK(c, gin.H{"menus": buildMenuTree(nodes)})
	})
}

// RegisterPlatformMenuCatalogRoute mounts GET /platform/menus — the COMPLETE
// platform console tree (unfiltered) for the role editor. Wire on the platform group.
func (a *App) RegisterPlatformMenuCatalogRoute(r *gin.RouterGroup) {
	r.GET("/platform/menus", func(c *gin.Context) {
		nodes := []module.MenuNode{}
		for _, n := range a.allMenus() {
			if consoleMatches(n, "platform") {
				nodes = append(nodes, n)
			}
		}
		apiresp.OK(c, gin.H{"menus": buildMenuTree(nodes)})
	})
}
