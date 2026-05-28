package app

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leo/iop/server/internal/config"
	"github.com/leo/iop/server/internal/infrastructure/health"
	loggerinfra "github.com/leo/iop/server/internal/infrastructure/logger"
	pginfra "github.com/leo/iop/server/internal/infrastructure/pg"
	iface "github.com/leo/iop/server/internal/interface"
	"github.com/leo/iop/server/internal/services/dictionary"
	"github.com/leo/iop/server/internal/services/localization"
	"github.com/leo/iop/server/internal/shared/eventbus"
	"github.com/leo/iop/server/internal/shared/tenantdb"
	"go.uber.org/zap"
)

// App holds every wired component. Owned by main(), shut down on signal.
type App struct {
	Cfg        *config.Config
	Logger     *zap.Logger
	Pool       *pgxpool.Pool
	Platform   *tenantdb.PlatformDB
	Tenant     *tenantdb.TenantDB
	Bus        *eventbus.InprocBus
	Health     *health.Registry
	Dictionary *dictionary.Service
	I18n       *localization.Service
}

// Build wires components in dependency order. Returns (*App, cleanup, error).
func Build(ctx context.Context, cfg *config.Config) (*App, func(), error) {
	logger, err := loggerinfra.New(loggerinfra.Config{Level: cfg.Logger.Level, Format: cfg.Logger.Format})
	if err != nil {
		return nil, nil, fmt.Errorf("logger: %w", err)
	}

	pool, err := pginfra.NewPool(ctx, pginfra.Config{DSN: cfg.DB.DSN})
	if err != nil {
		return nil, nil, fmt.Errorf("pg: %w", err)
	}

	bus := eventbus.NewInprocBus(4).WithLogger(logger)
	bus.Start()

	healthReg := health.NewRegistry()
	healthReg.Register(health.Check{
		Name: "pg", Critical: true,
		Check: func(c context.Context) error { return pool.Ping(c) },
	})

	dictSvc := dictionary.NewService(dictionary.MemoryRepo(map[string][]dictionary.Item{
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
	}))

	bundle, _ := localization.LoadYAMLBundle("./configs/i18n")
	if bundle == nil {
		bundle = localization.MapBundle(nil)
	}
	i18n := localization.NewService(bundle, "zh-CN")

	a := &App{
		Cfg:        cfg,
		Logger:     logger,
		Pool:       pool,
		Platform:   tenantdb.NewPlatformDB(pool),
		Tenant:     tenantdb.NewTenantDB(pool),
		Bus:        bus,
		Health:     healthReg,
		Dictionary: dictSvc,
		I18n:       i18n,
	}

	cleanup := func() {
		_ = bus.Close()
		pool.Close()
		_ = logger.Sync()
	}
	return a, cleanup, nil
}

// Engine returns the wired Gin engine with all routes mounted.
func (a *App) Engine() *gin.Engine {
	r := iface.New(iface.Config{AllowedOrigins: a.Cfg.Server.AllowedOrigins}, a.Logger, a.Health)
	api := r.Group("/api")
	dictionary.RegisterRoutes(api, a.Dictionary)
	return r
}
