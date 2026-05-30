package app

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	redislib "github.com/redis/go-redis/v9"

	"github.com/leo/iop/server/internal/config"
	"github.com/leo/iop/server/internal/infrastructure/health"
	loggerinfra "github.com/leo/iop/server/internal/infrastructure/logger"
	pginfra "github.com/leo/iop/server/internal/infrastructure/pg"
	redisinfra "github.com/leo/iop/server/internal/infrastructure/redis"
	"github.com/leo/iop/server/internal/contexts/crm"
	"github.com/leo/iop/server/internal/contexts/okr"
	okriface "github.com/leo/iop/server/internal/contexts/okr/interface"
	iface "github.com/leo/iop/server/internal/interface"
	"github.com/leo/iop/server/internal/interface/middleware"
	"github.com/leo/iop/server/internal/services/appstore"
	"github.com/leo/iop/server/internal/services/audit"
	"github.com/leo/iop/server/internal/services/dictionary"
	"github.com/leo/iop/server/internal/services/filestorage"
	"github.com/leo/iop/server/internal/services/iam"
	"github.com/leo/iop/server/internal/services/localization"
	"github.com/leo/iop/server/internal/services/notification"
	"github.com/leo/iop/server/internal/services/tenancy"
	"github.com/leo/iop/server/internal/shared/eventbus"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/module"
	"github.com/leo/iop/server/internal/shared/tenantdb"
	"go.uber.org/zap"
)

// App holds every wired component. Owned by main(), shut down on signal.
type App struct {
	Cfg        *config.Config
	Logger     *zap.Logger
	Pool       *pgxpool.Pool
	RDB        *redislib.Client
	Platform   *tenantdb.PlatformDB
	Tenant     *tenantdb.TenantDB
	Bus        *eventbus.InprocBus
	Health     *health.Registry
	Dictionary  *dictionary.Service
	DictMemory  dictionary.Repository
	I18n        *localization.Service
	Tenancy     *tenancy.Service
	IAM         *iam.Service
	Audit       *audit.Service
	Notif       *notification.Service
	FileStorage *filestorage.Service
	Modules     *module.Registry
	AppStore    *appstore.Service
}

// Build wires components in dependency order. Returns (*App, cleanup, error).
func Build(ctx context.Context, cfg *config.Config) (*App, func(), error) {
	logger, err := loggerinfra.New(loggerinfra.Config{Level: cfg.Logger.Level, Format: cfg.Logger.Format})
	if err != nil {
		return nil, nil, fmt.Errorf("logger: %w", err)
	}

	pool, err := pginfra.NewPool(ctx, pginfra.Config{DSN: cfg.DB.DSN, Logger: logger})
	if err != nil {
		return nil, nil, fmt.Errorf("pg: %w", err)
	}

	// Redis is optional in dev; if unavailable, we run in degraded mode.
	var rdb *redislib.Client
	if cfg.Redis.Addr != "" {
		rdb, err = redisinfra.New(ctx, redisinfra.Config{Addr: cfg.Redis.Addr})
		if err != nil {
			logger.Warn("redis unavailable, running in degraded mode", zap.Error(err))
			rdb = nil
		}
	}

	bus := eventbus.NewInprocBus(4).WithLogger(logger)
	bus.Start()

	clk := kernel.RealClock{}

	healthReg := health.NewRegistry()
	healthReg.Register(health.Check{
		Name: "pg", Critical: true,
		Check: func(c context.Context) error { return pool.Ping(c) },
	})
	if rdb != nil {
		healthReg.Register(health.Check{
			Name: "redis", Critical: false, // noncritical → degraded but ready
			Check: func(c context.Context) error { return rdb.Ping(c).Err() },
		})
	}

	dictMemory := dictionary.MemoryRepo(map[string][]dictionary.Item{
		"plan_level": {
			{TypeCode: "plan_level", Code: "year", Name: "年度", SortOrder: 1, Active: true},
			{TypeCode: "plan_level", Code: "half_year", Name: "半年", SortOrder: 2, Active: true},
			{TypeCode: "plan_level", Code: "month", Name: "月度", SortOrder: 3, Active: true},
			{TypeCode: "plan_level", Code: "week", Name: "周度", SortOrder: 4, Active: true},
		},
		"report_type": {
			{TypeCode: "report_type", Code: "daily", Name: "日报", SortOrder: 1, Active: true},
			{TypeCode: "report_type", Code: "weekly", Name: "周报", SortOrder: 2, Active: true},
		},
	})
	dictSvc := dictionary.NewService(dictMemory)

	bundle, _ := localization.LoadYAMLBundle("./configs/i18n")
	if bundle == nil {
		bundle = localization.MapBundle(nil)
	}
	i18n := localization.NewService(bundle, "zh-CN")

	// Tenancy
	provisioner := tenancy.NewSchemaProvisioner(pool, "./migrations/tenant_template")
	tenantSvc := tenancy.NewService(
		tenancy.NewPGTenantRepo(pool),
		tenancy.NewPGMembershipRepo(pool),
		provisioner, bus, clk,
	)

	// IAM
	secret := cfg.Auth.JWTSecret
	if secret == "" {
		secret = "dev-only-change-me-32-chars-min-len"
	}
	iamSvc := iam.NewService(
		iam.NewPGRepo(pool),
		iam.NewHS256Signer(secret),
		tenantSvc, rdb, bus, clk,
	)

	tenantDB := tenantdb.NewTenantDB(pool)

	// Audit: tenant lookup via tenancy service
	auditSvc := audit.NewService(tenantDB,
		func(c context.Context, id kernel.ID) (string, bool) {
			t, _ := tenantSvc.GetTenant(c, id)
			if t == nil {
				return "", false
			}
			return t.SchemaName, true
		}, logger)
	auditSvc.Subscribe(bus, []string{
		"tenancy.tenant_created", "tenancy.tenant_suspended", "tenancy.tenant_resumed",
		"tenancy.tenant_closed", "tenancy.member_joined",
		"iam.user_logged_in", "iam.user_logged_out", "iam.login_failed",
		"okr.plan_created", "okr.plan_item_completed", "okr.plan_closed",
		"okr.daily_submitted", "okr.weekly_submitted",
	})

	// Notification: subscribes to selected events
	notifSvc := notification.NewService(tenantDB, logger, clk)
	notifSvc.Wire(bus)

	// FileStorage (MinIO).  Best-effort init — if MinIO unreachable in dev, leave nil.
	var fsSvc *filestorage.Service
	fsSvc, fsErr := filestorage.NewService(ctx, tenantDB, filestorage.Config{
		Endpoint:  cfg.MinIO.Endpoint,
		AccessKey: cfg.MinIO.AccessKey,
		SecretKey: cfg.MinIO.SecretKey,
	}, clk)
	if fsErr != nil {
		logger.Warn("filestorage unavailable, file routes will return 503", zap.Error(fsErr))
		fsSvc = nil
	}

	// === Module Registry === — every business module ships a Module impl;
	// registering it here is the ONLY change needed to add a new app.
	registry := module.NewRegistry()
	platformDB := tenantdb.NewPlatformDB(pool)
	deps := module.Deps{
		Pool:     pool,
		Tenant:   tenantDB,
		Platform: platformDB,
		Bus:      bus,
		Logger:   logger,
		Clock:    clk,
	}
	registry.Register(okr.New(deps))
	registry.Register(crm.New(deps))
	// registry.Register(approval.New(deps))

	appStore := appstore.NewService(pool, registry, clk)

	a := &App{
		Cfg:         cfg,
		Logger:      logger,
		Pool:        pool,
		RDB:         rdb,
		Platform:    platformDB,
		Tenant:      tenantDB,
		Bus:         bus,
		Health:      healthReg,
		Dictionary:  dictSvc,
		DictMemory:  dictMemory,
		I18n:        i18n,
		Tenancy:     tenantSvc,
		IAM:         iamSvc,
		Audit:       auditSvc,
		Notif:       notifSvc,
		FileStorage: fsSvc,
		Modules:     registry,
		AppStore:    appStore,
	}

	cleanup := func() {
		_ = auditSvc.Close()
		_ = bus.Close()
		pool.Close()
		if rdb != nil {
			_ = rdb.Close()
		}
		_ = logger.Sync()
	}
	return a, cleanup, nil
}

// Engine returns the wired Gin engine with all routes mounted.
func (a *App) Engine() *gin.Engine {
	r := iface.New(iface.Config{AllowedOrigins: a.Cfg.Server.AllowedOrigins}, a.Logger, a.Health)
	r.Use(middleware.RateLimit(a.RDB, middleware.DefaultRateLimit()))
	r.Use(middleware.Idempotency(a.RDB))

	api := r.Group("/api")
	dictionary.RegisterRoutes(api, a.Dictionary)
	iam.RegisterRoutes(api, a.IAM, a.Tenancy)
	tenancy.RegisterRoutes(api, a.Tenancy, a.Pool)

	// Authenticated tenant-scoped group
	authT := api.Group("")
	authT.Use(iam.JWTAuth(a.IAM))
	authT.Use(iam.TenantLoader(a.Tenancy))
	audit.RegisterRoutes(authT, a.Audit)
	notification.RegisterRoutes(authT, a.Notif)
	if a.FileStorage != nil {
		filestorage.RegisterRoutes(authT, a.FileStorage)
	}

	// AppStore catalog (authenticated, tenant-scoped)
	appstore.RegisterCatalogRoutes(authT, a.AppStore)

	// Module routes — Registry mounts /api/apps/<code>/* for every module.
	// Each module also gets routes at the legacy /api/* paths via its own RegisterRoutes
	// for backward compatibility during transition. Modules can choose to wire either or both.
	deps := module.Deps{
		Pool:     a.Pool,
		Tenant:   a.Tenant,
		Platform: a.Platform,
		Bus:      a.Bus,
		Logger:   a.Logger,
		Clock:    kernel.RealClock{},
	}
	a.Modules.MountAll(authT, deps)
	// Backward-compat: also mount OKR at the flat /api/plans /api/reports /api/rollups paths
	// so the existing frontend keeps working without recompile.
	if okrMod, _ := a.Modules.Get("okr").(*okr.Module); okrMod != nil {
		okriface.RegisterRoutes(authT, okrMod.AppService())
	}

	// Personal /me routes — auth only (no admin gate)
	authOnly := api.Group("")
	authOnly.Use(iam.JWTAuth(a.IAM))
	iam.RegisterMeRoutes(authOnly, a.IAM)
	// /me/apps needs tenant context; mount under authT (after TenantLoader)
	appstore.RegisterMeRoutes(authT, a.AppStore)

	// Admin routes — require tenant_admin OR platform_admin (gate checks role grants)
	admin := authT.Group("")
	admin.Use(iam.TenantAdminRequired(a.IAM))
	tenancy.RegisterAdminRoutes(admin, a.Tenancy, a.Pool)
	iam.RegisterAdminRoutes(admin, a.IAM)
	dictionary.RegisterAdminRoutes(admin, dictionary.AdminConfig{
		Memory:   a.DictMemory,
		TenantDB: a.Tenant,
	}, []string{"plan_level", "report_type"})
	appstore.RegisterAdminRoutes(admin, a.AppStore)
	module.RegisterAdminRoutes(admin, a.Modules)

	return r
}
