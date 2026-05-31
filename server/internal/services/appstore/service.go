// Package appstore is the tenant-side enablement layer.
// Modules are code (installed by build); the AppStore decides which ones
// are visible/usable for each tenant.
package appstore

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/module"
)

// Installation is one (tenant, app) record from public.tenant_app.
type Installation struct {
	TenantID    kernel.ID  `json:"tenant_id"`
	AppCode     string     `json:"app_code"`
	InstalledAt time.Time  `json:"installed_at"`
	InstalledBy *kernel.ID `json:"installed_by,omitempty"`
}

// CatalogEntry pairs a module's manifest with an "installed" flag for the current tenant.
type CatalogEntry struct {
	module.Manifest
	Installed bool `json:"installed"`
}

type AppCategory struct {
	Code  string `json:"code"`
	Name  string `json:"name"`
	Color string `json:"color"`
	Order int    `json:"order"`
}

var defaultCategories = []AppCategory{
	{Code: "work", Name: "工作协同", Color: "var(--cat-goal)", Order: 10},
	{Code: "knowledge", Name: "知识内容", Color: "var(--cat-knowledge)", Order: 20},
	{Code: "data", Name: "数据工具", Color: "var(--cat-form)", Order: 30},
	{Code: "business", Name: "业务管理", Color: "var(--cat-biz)", Order: 40},
	{Code: "system", Name: "系统管理", Color: "var(--cat-admin)", Order: 50},
	{Code: "other", Name: "其他", Color: "var(--text-4)", Order: 90},
}

var defaultCategoryByApp = map[string]string{
	"okr":      "工作协同",
	"tasks":    "工作协同",
	"project":  "工作协同",
	"approval": "工作协同",
	"docs":     "知识内容",
	"news":     "知识内容",
	"books":    "知识内容",
	"mindmap":  "知识内容",
	"lcform":   "数据工具",
}

const userWorkspaceMarkerCode = "__workspace_initialized__"

type Service struct {
	pool     *pgxpool.Pool
	registry *module.Registry
	clock    kernel.Clock
}

func NewService(pool *pgxpool.Pool, registry *module.Registry, clk kernel.Clock) *Service {
	return &Service{pool: pool, registry: registry, clock: clk}
}

func Categories() []AppCategory {
	out := make([]AppCategory, len(defaultCategories))
	copy(out, defaultCategories)
	return out
}

// Catalog returns every module the binary supports, marked installed/not for the tenant.
func (s *Service) Catalog(ctx context.Context, tenantID kernel.ID) ([]CatalogEntry, error) {
	installed, err := s.listInstalled(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	displayConfig, err := s.loadDisplayConfig(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	manifests := s.registry.Manifests()
	out := make([]CatalogEntry, 0, len(manifests))
	for _, m := range manifests {
		m = s.displayManifest(m, displayConfig)
		out = append(out, CatalogEntry{
			Manifest:  m,
			Installed: installed[m.Code],
		})
	}
	return out, nil
}

// MyApps returns the ordered app codes for a user's personal workspace in a
// tenant. If the user has curated their own list (public.user_app rows), those
// are returned ordered by order_num. Otherwise it DEFAULTS to the tenant's
// installed apps (public.tenant_app) so users who never customized still see
// the org's apps. Codes are filtered to apps still registered in the binary.
func (s *Service) MyApps(ctx context.Context, platformUserID, tenantID kernel.ID) ([]string, error) {
	userCodes, err := s.ListUserApps(ctx, platformUserID, tenantID)
	if err != nil {
		return nil, err
	}
	initialized, err := s.hasWorkspaceRows(ctx, platformUserID, tenantID)
	if err != nil {
		return nil, err
	}
	codes := userCodes
	if len(codes) == 0 && !initialized {
		// Default to the tenant's installed apps, ordered by the registry's
		// stable Manifest order so the rail is deterministic.
		installed, err := s.listInstalled(ctx, tenantID)
		if err != nil {
			return nil, err
		}
		for _, m := range s.registry.Manifests() {
			if installed[m.Code] {
				codes = append(codes, m.Code)
			}
		}
	}
	// Drop codes for modules no longer in the binary.
	out := make([]string, 0, len(codes))
	for _, code := range codes {
		if s.registry.Get(code) != nil {
			out = append(out, code)
		}
	}
	return out, nil
}

// ListUserApps returns a user's curated app codes for a tenant, ordered by
// order_num. Empty slice (not nil error) when the user has no rows.
func (s *Service) ListUserApps(ctx context.Context, platformUserID, tenantID kernel.ID) ([]string, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT app_code FROM public.user_app
		 WHERE platform_user_id = $1 AND tenant_id = $2
		   AND app_code <> $3
		 ORDER BY order_num, app_code`,
		platformUserID, tenantID, userWorkspaceMarkerCode)
	if err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "appstore.list_user_apps_failed", "读取个人应用失败", err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		out = append(out, code)
	}
	return out, rows.Err()
}

// AddUserApp appends an app to the user's workspace (order_num = max+1).
// Idempotent: a second add is a no-op (keeps the existing position).
func (s *Service) AddUserApp(ctx context.Context, platformUserID, tenantID kernel.ID, appCode string) error {
	if s.registry.Get(appCode) == nil {
		return errors.New(errors.KindNotFound, "appstore.unknown_app",
			"应用未在本平台注册: "+appCode)
	}
	if err := s.materializeWorkspaceIfEmpty(ctx, platformUserID, tenantID); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO public.user_app (platform_user_id, tenant_id, app_code, order_num, added_at)
		 VALUES ($1, $2, $3,
		         COALESCE((SELECT max(order_num) + 1 FROM public.user_app
		                   WHERE platform_user_id = $1 AND tenant_id = $2), 0),
		         $4)
		 ON CONFLICT (platform_user_id, tenant_id, app_code) DO NOTHING`,
		platformUserID, tenantID, appCode, s.clock.Now())
	if err != nil {
		return errors.Wrap(errors.KindDatabase, "appstore.add_user_app_failed", "添加个人应用失败", err)
	}
	return nil
}

// RemoveUserApp removes an app from the user's workspace. No-op if absent.
func (s *Service) RemoveUserApp(ctx context.Context, platformUserID, tenantID kernel.ID, appCode string) error {
	if err := s.materializeWorkspaceIfEmpty(ctx, platformUserID, tenantID); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx,
		`DELETE FROM public.user_app
		 WHERE platform_user_id = $1 AND tenant_id = $2 AND app_code = $3`,
		platformUserID, tenantID, appCode)
	if err != nil {
		return errors.Wrap(errors.KindDatabase, "appstore.remove_user_app_failed", "移除个人应用失败", err)
	}
	return s.markWorkspaceInitialized(ctx, platformUserID, tenantID)
}

// SetUserAppOrder rewrites the user's workspace ordering from the given code
// list (order_num = index). Codes not registered in the binary are skipped.
// Runs in a transaction so the rewrite is atomic.
func (s *Service) SetUserAppOrder(ctx context.Context, platformUserID, tenantID kernel.ID, codes []string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return errors.Wrap(errors.KindDatabase, "appstore.order_failed", "排序失败", err)
	}
	defer tx.Rollback(ctx)
	now := s.clock.Now()
	order := 0
	for _, code := range codes {
		if s.registry.Get(code) == nil {
			continue
		}
		_, err := tx.Exec(ctx,
			`INSERT INTO public.user_app (platform_user_id, tenant_id, app_code, order_num, added_at)
			 VALUES ($1, $2, $3, $4, $5)
			 ON CONFLICT (platform_user_id, tenant_id, app_code)
			 DO UPDATE SET order_num = EXCLUDED.order_num`,
			platformUserID, tenantID, code, order, now)
		if err != nil {
			return errors.Wrap(errors.KindDatabase, "appstore.order_failed", "排序失败", err)
		}
		order++
	}
	if err := tx.Commit(ctx); err != nil {
		return errors.Wrap(errors.KindDatabase, "appstore.order_failed", "排序失败", err)
	}
	return nil
}

// IsEnabledForUser reports whether an app is usable for a given user in a
// tenant: true if the user added it (public.user_app) OR the tenant has it
// installed (public.tenant_app). The tenant-level OR keeps existing behavior
// working for users who never customized their workspace.
func (s *Service) IsEnabledForUser(ctx context.Context, tenantID, platformUserID kernel.ID, appCode string) (bool, error) {
	var n int
	err := s.pool.QueryRow(ctx,
		`SELECT
		   (SELECT count(*) FROM public.user_app
		      WHERE platform_user_id = $1 AND tenant_id = $2 AND app_code = $3)
		 + (SELECT count(*) FROM public.tenant_app
		      WHERE tenant_id = $2 AND app_code = $3)`,
		platformUserID, tenantID, appCode).Scan(&n)
	if err != nil {
		return false, errors.Wrap(errors.KindDatabase, "appstore.is_enabled_for_user_failed", "校验应用启用失败", err)
	}
	return n > 0, nil
}

// ManifestsForTenant returns the manifests for the given ordered codes, preserving
// order and dropping any code not registered in the binary.
func (s *Service) ManifestsForTenant(ctx context.Context, tenantID kernel.ID, codes []string) ([]module.Manifest, error) {
	displayConfig, err := s.loadDisplayConfig(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]module.Manifest, 0, len(codes))
	for _, code := range codes {
		if m := s.registry.Get(code); m != nil {
			out = append(out, s.displayManifest(m.Manifest(), displayConfig))
		}
	}
	return out, nil
}

func (s *Service) SetCategory(ctx context.Context, tenantID kernel.ID, appCode string, category string, updatedBy kernel.ID) (module.Manifest, error) {
	mod := s.registry.Get(appCode)
	if mod == nil {
		return module.Manifest{}, errors.New(errors.KindNotFound, "appstore.unknown_app",
			"应用未在本平台注册: "+appCode)
	}
	normalized, err := normalizeCategory(category)
	if err != nil {
		return module.Manifest{}, err
	}
	if normalized == "" {
		_, err = s.pool.Exec(ctx,
			`DELETE FROM public.tenant_app_display_config
			 WHERE tenant_id = $1 AND app_code = $2`,
			tenantID, appCode)
	} else {
		_, err = s.pool.Exec(ctx,
			`INSERT INTO public.tenant_app_display_config
			   (tenant_id, app_code, category, updated_at, updated_by)
			 VALUES ($1, $2, $3, $4, $5)
			 ON CONFLICT (tenant_id, app_code)
			 DO UPDATE SET category = EXCLUDED.category,
			               updated_at = EXCLUDED.updated_at,
			               updated_by = EXCLUDED.updated_by`,
			tenantID, appCode, normalized, s.clock.Now(), updatedBy)
	}
	if err != nil {
		return module.Manifest{}, errors.Wrap(errors.KindDatabase, "appstore.set_category_failed", "保存应用分类失败", err)
	}
	cfg, err := s.loadDisplayConfig(ctx, tenantID)
	if err != nil {
		return module.Manifest{}, err
	}
	return s.displayManifest(mod.Manifest(), cfg), nil
}

// Install adds an enablement row. No-op if already present.
func (s *Service) Install(ctx context.Context, tenantID kernel.ID, appCode string, installedBy kernel.ID) error {
	mod := s.registry.Get(appCode)
	if mod == nil {
		return errors.New(errors.KindNotFound, "appstore.unknown_app",
			"应用未在本平台注册: "+appCode)
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO public.tenant_app (tenant_id, app_code, installed_at, installed_by)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (tenant_id, app_code) DO NOTHING`,
		tenantID, appCode, s.clock.Now(), installedBy)
	if err != nil {
		return errors.Wrap(errors.KindDatabase, "appstore.install_failed", "安装失败", err)
	}
	// Run the module's optional install hook; roll back the enablement on failure
	// so install is all-or-nothing.
	if hook, ok := mod.(module.InstallHook); ok {
		if err := hook.OnInstall(ctx, tenantID); err != nil {
			_, _ = s.pool.Exec(ctx,
				`DELETE FROM public.tenant_app WHERE tenant_id = $1 AND app_code = $2`,
				tenantID, appCode)
			return errors.Wrap(errors.KindInternal, "appstore.install_hook_failed", "应用初始化失败", err)
		}
	}
	return nil
}

// Uninstall removes an enablement row. No-op if absent. Runs the module's
// optional uninstall hook first (best-effort cleanup).
func (s *Service) Uninstall(ctx context.Context, tenantID kernel.ID, appCode string) error {
	if mod := s.registry.Get(appCode); mod != nil {
		if hook, ok := mod.(module.UninstallHook); ok {
			_ = hook.OnUninstall(ctx, tenantID)
		}
	}
	_, err := s.pool.Exec(ctx,
		`DELETE FROM public.tenant_app WHERE tenant_id = $1 AND app_code = $2`,
		tenantID, appCode)
	if err != nil {
		return errors.Wrap(errors.KindDatabase, "appstore.uninstall_failed", "卸载失败", err)
	}
	return nil
}

// IsInstalled returns true if the tenant has the app enabled.
func (s *Service) IsInstalled(ctx context.Context, tenantID kernel.ID, appCode string) (bool, error) {
	var n int
	err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM public.tenant_app WHERE tenant_id = $1 AND app_code = $2`,
		tenantID, appCode).Scan(&n)
	return n > 0, err
}

func (s *Service) listInstalled(ctx context.Context, tenantID kernel.ID) (map[string]bool, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT app_code FROM public.tenant_app WHERE tenant_id = $1`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		out[code] = true
	}
	return out, rows.Err()
}

func (s *Service) materializeWorkspaceIfEmpty(ctx context.Context, platformUserID, tenantID kernel.ID) error {
	initialized, err := s.hasWorkspaceRows(ctx, platformUserID, tenantID)
	if err != nil {
		return err
	}
	if initialized {
		return nil
	}
	installed, err := s.listInstalled(ctx, tenantID)
	if err != nil {
		return err
	}
	now := s.clock.Now()
	order := 0
	for _, m := range s.registry.Manifests() {
		if !installed[m.Code] {
			continue
		}
		_, err := s.pool.Exec(ctx,
			`INSERT INTO public.user_app
			   (platform_user_id, tenant_id, app_code, order_num, added_at)
			 VALUES ($1, $2, $3, $4, $5)
			 ON CONFLICT (platform_user_id, tenant_id, app_code) DO NOTHING`,
			platformUserID, tenantID, m.Code, order, now)
		if err != nil {
			return errors.Wrap(errors.KindDatabase, "appstore.materialize_user_apps_failed", "初始化个人应用失败", err)
		}
		order++
	}
	return s.markWorkspaceInitialized(ctx, platformUserID, tenantID)
}

func (s *Service) hasWorkspaceRows(ctx context.Context, platformUserID, tenantID kernel.ID) (bool, error) {
	var n int
	err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM public.user_app
		  WHERE platform_user_id = $1 AND tenant_id = $2`,
		platformUserID, tenantID).Scan(&n)
	if err != nil {
		return false, errors.Wrap(errors.KindDatabase, "appstore.workspace_state_failed", "读取个人应用状态失败", err)
	}
	return n > 0, nil
}

func (s *Service) markWorkspaceInitialized(ctx context.Context, platformUserID, tenantID kernel.ID) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO public.user_app
		   (platform_user_id, tenant_id, app_code, order_num, added_at)
		 VALUES ($1, $2, $3, -1, $4)
		 ON CONFLICT (platform_user_id, tenant_id, app_code) DO NOTHING`,
		platformUserID, tenantID, userWorkspaceMarkerCode, s.clock.Now())
	if err != nil {
		return errors.Wrap(errors.KindDatabase, "appstore.mark_workspace_failed", "保存个人应用状态失败", err)
	}
	return nil
}

func (s *Service) loadDisplayConfig(ctx context.Context, tenantID kernel.ID) (map[string]string, error) {
	out := map[string]string{}
	if tenantID == "" {
		return out, nil
	}
	rows, err := s.pool.Query(ctx,
		`SELECT app_code, category
		   FROM public.tenant_app_display_config
		  WHERE tenant_id = $1`,
		tenantID)
	if err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "appstore.load_display_config_failed", "读取应用展示配置失败", err)
	}
	defer rows.Close()
	for rows.Next() {
		var code, category string
		if err := rows.Scan(&code, &category); err != nil {
			return nil, err
		}
		if cat := strings.TrimSpace(category); cat != "" {
			out[code] = cat
		}
	}
	return out, rows.Err()
}

func (s *Service) displayManifest(m module.Manifest, displayConfig map[string]string) module.Manifest {
	m.Category = defaultCategory(m.Code, m.Category)
	if category := strings.TrimSpace(displayConfig[m.Code]); category != "" {
		m.Category = category
	}
	return m
}

func defaultCategory(appCode, fallback string) string {
	if category := defaultCategoryByApp[appCode]; category != "" {
		return category
	}
	if category := strings.TrimSpace(fallback); category != "" {
		return category
	}
	return "其他"
}

func normalizeCategory(category string) (string, error) {
	category = strings.TrimSpace(category)
	if category == "" {
		return "", nil
	}
	for _, c := range defaultCategories {
		if category == c.Name || category == c.Code {
			return c.Name, nil
		}
	}
	valid := make([]string, 0, len(defaultCategories))
	for _, c := range defaultCategories {
		valid = append(valid, c.Name)
	}
	slices.Sort(valid)
	return "", errors.New(errors.KindParam, "appstore.invalid_category",
		"应用分类必须是: "+strings.Join(valid, "、"))
}
