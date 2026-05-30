// Package appstore is the tenant-side enablement layer.
// Modules are code (installed by build); the AppStore decides which ones
// are visible/usable for each tenant.
package appstore

import (
	"context"
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

type Service struct {
	pool     *pgxpool.Pool
	registry *module.Registry
	clock    kernel.Clock
}

func NewService(pool *pgxpool.Pool, registry *module.Registry, clk kernel.Clock) *Service {
	return &Service{pool: pool, registry: registry, clock: clk}
}

// Catalog returns every module the binary supports, marked installed/not for the tenant.
func (s *Service) Catalog(ctx context.Context, tenantID kernel.ID) ([]CatalogEntry, error) {
	installed, err := s.listInstalled(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	manifests := s.registry.Manifests()
	out := make([]CatalogEntry, 0, len(manifests))
	for _, m := range manifests {
		out = append(out, CatalogEntry{
			Manifest:  m,
			Installed: installed[m.Code],
		})
	}
	return out, nil
}

// MyApps returns the subset of installed app manifests for a tenant.
// This is what the left rail uses.
func (s *Service) MyApps(ctx context.Context, tenantID kernel.ID) ([]module.Manifest, error) {
	installed, err := s.listInstalled(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := []module.Manifest{}
	for _, m := range s.registry.Manifests() {
		if installed[m.Code] {
			out = append(out, m)
		}
	}
	return out, nil
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
