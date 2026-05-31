package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/module"
)

const (
	menuStatusActive   = "active"
	menuStatusDisabled = "disabled"
)

var validMenuTypes = map[string]bool{
	"dir": true, "menu": true, "button": true, "link": true, "iframe": true, "micro": true,
}

var validMenuConsoles = map[string]bool{
	"platform": true, "tenant": true, "both": true,
}

type menuCatalogFilter struct {
	Console         string
	IncludeDeleted  bool
	IncludeDisabled bool
	OnlyVisible     bool
	Search          string
}

type menuConfigCmd struct {
	Key          string
	Title        string
	Type         string
	Parent       string
	Path         string
	Component    string
	Perm         string
	Icon         string
	Order        int
	Visible      bool
	Cacheable    bool
	Status       string
	App          string
	Console      string
	ExternalURL  string
	IframeURL    string
	MicroAppCode string
	MicroEntry   string
}

func normalizeStaticMenuNode(n module.MenuNode) module.MenuNode {
	if n.Type == "" {
		n.Type = "menu"
	}
	if n.Console == "" {
		n.Console = "tenant"
	}
	if n.Status == "" {
		n.Status = menuStatusActive
	}
	n.Visible = true
	n.BuiltIn = true
	return n
}

func (a *App) codeMenus() []module.MenuNode {
	out := append([]module.MenuNode{}, builtinMenus()...)
	out = append(out, a.Modules.AllMenus()...)
	for i := range out {
		out[i] = normalizeStaticMenuNode(out[i])
	}
	return out
}

func (a *App) syncMenuCatalog(ctx context.Context) error {
	for _, n := range a.codeMenus() {
		if n.Key == "" || n.Title == "" {
			continue
		}
		_, err := a.Pool.Exec(ctx,
			`INSERT INTO public.menu_catalog (
			    id, menu_key, title, menu_type, parent_key, route_path, component_path,
			    permission_code, icon, order_num, visible, cacheable, status, app_code,
			    console, external_url, iframe_url, micro_app_code, micro_entry, is_builtin, source
			 ) VALUES (
			    $1,$2,$3,$4,NULLIF($5,''),NULLIF($6,''),NULLIF($7,''),
			    NULLIF($8,''),NULLIF($9,''),$10,TRUE,$11,'active',NULLIF($12,''),
			    $13,NULLIF($14,''),NULLIF($15,''),NULLIF($16,''),NULLIF($17,''),TRUE,'code'
			 )
			 ON CONFLICT (menu_key) DO UPDATE SET
			    title = EXCLUDED.title,
			    menu_type = EXCLUDED.menu_type,
			    parent_key = EXCLUDED.parent_key,
			    route_path = EXCLUDED.route_path,
			    component_path = EXCLUDED.component_path,
			    permission_code = EXCLUDED.permission_code,
			    icon = EXCLUDED.icon,
			    order_num = EXCLUDED.order_num,
			    cacheable = EXCLUDED.cacheable,
			    app_code = EXCLUDED.app_code,
			    console = EXCLUDED.console,
			    external_url = EXCLUDED.external_url,
			    iframe_url = EXCLUDED.iframe_url,
			    micro_app_code = EXCLUDED.micro_app_code,
			    micro_entry = EXCLUDED.micro_entry,
			    is_builtin = TRUE,
			    source = 'code',
			    updated_at = now()`,
			kernel.NewID(), n.Key, n.Title, n.Type, n.Parent, n.Path, n.Component,
			n.Perm, n.Icon, n.Order, n.Cacheable, n.App, n.Console,
			n.ExternalURL, n.IframeURL, n.MicroAppCode, n.MicroEntry)
		if err != nil {
			return errors.Wrap(errors.KindDatabase, "menu.sync_failed", "同步菜单目录失败", err)
		}
	}
	return nil
}

func (a *App) loadMenuCatalog(ctx context.Context, filter menuCatalogFilter) ([]module.MenuNode, error) {
	where := []string{"deleted_at IS NULL"}
	args := []any{}
	idx := 1
	if filter.IncludeDeleted {
		where = []string{"1=1"}
	}
	if filter.Console != "" {
		where = append(where, fmt.Sprintf("(console = $%d OR console = 'both')", idx))
		args = append(args, filter.Console)
		idx++
	}
	if !filter.IncludeDisabled {
		where = append(where, "status = 'active'")
	}
	if filter.OnlyVisible {
		where = append(where, "visible = TRUE")
	}
	if q := strings.TrimSpace(filter.Search); q != "" {
		where = append(where, fmt.Sprintf("(menu_key ILIKE $%d OR title ILIKE $%d OR COALESCE(permission_code,'') ILIKE $%d)", idx, idx, idx))
		args = append(args, "%"+q+"%")
		idx++
	}
	rows, err := a.Pool.Query(ctx,
		`SELECT id, menu_key, title, COALESCE(icon,''), COALESCE(route_path,''),
		        COALESCE(component_path,''), COALESCE(parent_key,''), menu_type, console,
		        COALESCE(app_code,''), COALESCE(permission_code,''), order_num, visible,
		        cacheable, status, COALESCE(external_url,''), COALESCE(iframe_url,''),
		        COALESCE(micro_app_code,''), COALESCE(micro_entry,''), is_builtin
		 FROM public.menu_catalog
		 WHERE `+strings.Join(where, " AND ")+`
		 ORDER BY order_num, menu_key`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []module.MenuNode{}
	for rows.Next() {
		var n module.MenuNode
		if err := rows.Scan(&n.ID, &n.Key, &n.Title, &n.Icon, &n.Path, &n.Component, &n.Parent,
			&n.Type, &n.Console, &n.App, &n.Perm, &n.Order, &n.Visible, &n.Cacheable, &n.Status,
			&n.ExternalURL, &n.IframeURL, &n.MicroAppCode, &n.MicroEntry, &n.BuiltIn); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (a *App) allMenus(ctx context.Context, filter menuCatalogFilter) []module.MenuNode {
	nodes, err := a.loadMenuCatalog(ctx, filter)
	if err == nil {
		return nodes
	}
	a.Logger.Warn("load menu_catalog failed; falling back to code menus")
	out := []module.MenuNode{}
	for _, n := range a.codeMenus() {
		if filter.Console != "" && !consoleMatches(n, filter.Console) {
			continue
		}
		if !filter.IncludeDisabled && n.Status != menuStatusActive {
			continue
		}
		if filter.OnlyVisible && !n.Visible {
			continue
		}
		out = append(out, n)
	}
	return out
}

func normalizeMenuCmd(cmd menuConfigCmd, partial bool) (menuConfigCmd, error) {
	cmd.Key = strings.TrimSpace(cmd.Key)
	cmd.Title = strings.TrimSpace(cmd.Title)
	cmd.Type = strings.TrimSpace(cmd.Type)
	cmd.Parent = strings.TrimSpace(cmd.Parent)
	cmd.Path = strings.TrimSpace(cmd.Path)
	cmd.Component = strings.TrimSpace(cmd.Component)
	cmd.Perm = strings.TrimSpace(cmd.Perm)
	cmd.Icon = strings.TrimSpace(cmd.Icon)
	cmd.Status = strings.TrimSpace(cmd.Status)
	cmd.App = strings.TrimSpace(cmd.App)
	cmd.Console = strings.TrimSpace(cmd.Console)
	cmd.ExternalURL = strings.TrimSpace(cmd.ExternalURL)
	cmd.IframeURL = strings.TrimSpace(cmd.IframeURL)
	cmd.MicroAppCode = strings.TrimSpace(cmd.MicroAppCode)
	cmd.MicroEntry = strings.TrimSpace(cmd.MicroEntry)
	if !partial {
		if cmd.Key == "" || cmd.Title == "" {
			return cmd, errors.New(errors.KindParam, "menu.invalid", "菜单标识和名称必填")
		}
	}
	if cmd.Type == "" {
		cmd.Type = "menu"
	}
	if !validMenuTypes[cmd.Type] {
		return cmd, errors.New(errors.KindParam, "menu.invalid_type", "菜单类型非法")
	}
	if cmd.Console == "" {
		cmd.Console = "tenant"
	}
	if !validMenuConsoles[cmd.Console] {
		return cmd, errors.New(errors.KindParam, "menu.invalid_console", "租户类型非法")
	}
	if cmd.Status == "" {
		cmd.Status = menuStatusActive
	}
	if cmd.Status != menuStatusActive && cmd.Status != menuStatusDisabled {
		return cmd, errors.New(errors.KindParam, "menu.invalid_status", "菜单状态非法")
	}
	if cmd.Type == "button" && cmd.Perm == "" {
		return cmd, errors.New(errors.KindParam, "menu.button_perm_required", "按钮必须配置权限标识")
	}
	return cmd, nil
}

func (a *App) parentExists(ctx context.Context, parent string) (bool, error) {
	if parent == "" {
		return true, nil
	}
	var exists bool
	err := a.Pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM public.menu_catalog WHERE menu_key = $1 AND deleted_at IS NULL)`,
		parent).Scan(&exists)
	return exists, err
}

func (a *App) wouldCreateMenuCycle(ctx context.Context, key, parent string) (bool, error) {
	for parent != "" {
		if parent == key {
			return true, nil
		}
		var next string
		err := a.Pool.QueryRow(ctx,
			`SELECT COALESCE(parent_key,'') FROM public.menu_catalog
			 WHERE menu_key = $1 AND deleted_at IS NULL`, parent).Scan(&next)
		if err == pgx.ErrNoRows {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		parent = next
	}
	return false, nil
}

func (a *App) createMenu(ctx context.Context, cmd menuConfigCmd) (module.MenuNode, error) {
	cmd, err := normalizeMenuCmd(cmd, false)
	if err != nil {
		return module.MenuNode{}, err
	}
	if ok, err := a.parentExists(ctx, cmd.Parent); err != nil {
		return module.MenuNode{}, err
	} else if !ok {
		return module.MenuNode{}, errors.New(errors.KindParam, "menu.parent_not_found", "父级菜单不存在")
	}
	id := kernel.NewID()
	_, err = a.Pool.Exec(ctx,
		`INSERT INTO public.menu_catalog (
		    id, menu_key, title, menu_type, parent_key, route_path, component_path,
		    permission_code, icon, order_num, visible, cacheable, status, app_code,
		    console, external_url, iframe_url, micro_app_code, micro_entry, is_builtin, source
		 ) VALUES ($1,$2,$3,$4,NULLIF($5,''),NULLIF($6,''),NULLIF($7,''),NULLIF($8,''),
		           NULLIF($9,''),$10,$11,$12,$13,NULLIF($14,''),$15,NULLIF($16,''),
		           NULLIF($17,''),NULLIF($18,''),NULLIF($19,''),FALSE,'manual')`,
		id, cmd.Key, cmd.Title, cmd.Type, cmd.Parent, cmd.Path, cmd.Component, cmd.Perm,
		cmd.Icon, cmd.Order, cmd.Visible, cmd.Cacheable, cmd.Status, cmd.App, cmd.Console,
		cmd.ExternalURL, cmd.IframeURL, cmd.MicroAppCode, cmd.MicroEntry)
	if err != nil {
		return module.MenuNode{}, errors.Wrap(errors.KindDatabase, "menu.create_failed", "创建菜单失败", err)
	}
	return a.getMenuByKey(ctx, cmd.Key)
}

func (a *App) getMenuByKey(ctx context.Context, key string) (module.MenuNode, error) {
	nodes, err := a.loadMenuCatalog(ctx, menuCatalogFilter{IncludeDisabled: true})
	if err != nil {
		return module.MenuNode{}, err
	}
	for _, n := range nodes {
		if n.Key == key {
			return n, nil
		}
	}
	return module.MenuNode{}, errors.New(errors.KindNotFound, "menu.not_found", "菜单不存在")
}

func (a *App) updateMenu(ctx context.Context, key string, cmd menuConfigCmd) error {
	cur, err := a.getMenuByKey(ctx, key)
	if err != nil {
		return err
	}
	if cmd.Key == "" {
		cmd.Key = cur.Key
	}
	if cmd.Title == "" {
		cmd.Title = cur.Title
	}
	if cmd.Type == "" {
		cmd.Type = cur.Type
	}
	if cmd.Console == "" {
		cmd.Console = cur.Console
	}
	if cmd.Status == "" {
		cmd.Status = cur.Status
	}
	cmd, err = normalizeMenuCmd(cmd, true)
	if err != nil {
		return err
	}
	if ok, err := a.parentExists(ctx, cmd.Parent); err != nil {
		return err
	} else if !ok {
		return errors.New(errors.KindParam, "menu.parent_not_found", "父级菜单不存在")
	}
	if cycle, err := a.wouldCreateMenuCycle(ctx, key, cmd.Parent); err != nil {
		return err
	} else if cycle {
		return errors.New(errors.KindParam, "menu.cycle", "不能把菜单移动到自己的子级下")
	}
	_, err = a.Pool.Exec(ctx,
		`UPDATE public.menu_catalog
		    SET menu_key = $2, title = $3, menu_type = $4, parent_key = NULLIF($5,''),
		        route_path = NULLIF($6,''), component_path = NULLIF($7,''),
		        permission_code = NULLIF($8,''), icon = NULLIF($9,''), order_num = $10,
		        visible = $11, cacheable = $12, status = $13, app_code = NULLIF($14,''),
		        console = $15, external_url = NULLIF($16,''), iframe_url = NULLIF($17,''),
		        micro_app_code = NULLIF($18,''), micro_entry = NULLIF($19,''), updated_at = now()
		  WHERE menu_key = $1 AND deleted_at IS NULL`,
		key, cmd.Key, cmd.Title, cmd.Type, cmd.Parent, cmd.Path, cmd.Component, cmd.Perm,
		cmd.Icon, cmd.Order, cmd.Visible, cmd.Cacheable, cmd.Status, cmd.App, cmd.Console,
		cmd.ExternalURL, cmd.IframeURL, cmd.MicroAppCode, cmd.MicroEntry)
	if err != nil {
		return errors.Wrap(errors.KindDatabase, "menu.update_failed", "更新菜单失败", err)
	}
	return nil
}

func (a *App) deleteMenu(ctx context.Context, key string) error {
	var builtin bool
	var children int
	if err := a.Pool.QueryRow(ctx,
		`SELECT is_builtin FROM public.menu_catalog WHERE menu_key = $1 AND deleted_at IS NULL`, key).Scan(&builtin); err != nil {
		return errors.New(errors.KindNotFound, "menu.not_found", "菜单不存在")
	}
	if builtin {
		return errors.New(errors.KindForbidden, "menu.builtin_delete_forbidden", "内置菜单不能删除")
	}
	if err := a.Pool.QueryRow(ctx,
		`SELECT count(*) FROM public.menu_catalog WHERE parent_key = $1 AND deleted_at IS NULL`, key).Scan(&children); err != nil {
		return err
	}
	if children > 0 {
		return errors.New(errors.KindConflict, "menu.has_children", "存在子菜单，不能删除")
	}
	_, err := a.Pool.Exec(ctx,
		`UPDATE public.menu_catalog SET deleted_at = now(), status = 'disabled' WHERE menu_key = $1 AND deleted_at IS NULL`, key)
	return err
}

func (a *App) tenantMenuOverrides(ctx context.Context, tenantID kernel.ID) (map[string]bool, map[string]int, error) {
	rows, err := a.Pool.Query(ctx,
		`SELECT c.menu_key, tm.enabled, COALESCE(tm.order_num, c.order_num)
		 FROM public.tenant_menu tm
		 JOIN public.menu_catalog c ON c.id = tm.menu_id
		 WHERE tm.tenant_id = $1 AND c.deleted_at IS NULL`, tenantID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	enabled := map[string]bool{}
	orders := map[string]int{}
	for rows.Next() {
		var key string
		var on bool
		var order int
		if err := rows.Scan(&key, &on, &order); err != nil {
			return nil, nil, err
		}
		enabled[key] = on
		orders[key] = order
	}
	return enabled, orders, rows.Err()
}

func (a *App) setTenantMenu(ctx context.Context, tenantID kernel.ID, key string, enabled bool, order *int) error {
	var id kernel.ID
	var console string
	if err := a.Pool.QueryRow(ctx,
		`SELECT id, console FROM public.menu_catalog WHERE menu_key = $1 AND deleted_at IS NULL`, key).Scan(&id, &console); err != nil {
		return errors.New(errors.KindNotFound, "menu.not_found", "菜单不存在")
	}
	if console == "platform" {
		return errors.New(errors.KindForbidden, "menu.platform_forbidden", "租户管理员不能配置平台级菜单")
	}
	_, err := a.Pool.Exec(ctx,
		`INSERT INTO public.tenant_menu (tenant_id, menu_id, enabled, order_num, updated_at)
		 VALUES ($1,$2,$3,$4,now())
		 ON CONFLICT (tenant_id, menu_id)
		 DO UPDATE SET enabled = EXCLUDED.enabled,
		               order_num = COALESCE(EXCLUDED.order_num, public.tenant_menu.order_num),
		               updated_at = now()`,
		tenantID, id, enabled, order)
	return err
}
