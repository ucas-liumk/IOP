package iface

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leo/iop/server/internal/infrastructure/health"
	"github.com/leo/iop/server/internal/infrastructure/metrics"
	"github.com/leo/iop/server/internal/interface/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// Version is set at build time via -ldflags "-X main.Version=...".
var Version = "dev"

// Config carries runtime knobs from configs/dev.yaml via app DI.
type Config struct {
	AllowedOrigins []string
}

// New wires the Gin engine with M1 middleware chain + system endpoints.
// services/* will register their own routes via the returned *gin.Engine.
func New(cfg Config, logger *zap.Logger, healthReg *health.Registry) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// IMPORTANT: Recover must be first so panics inside other middleware are caught.
	r.Use(middleware.Recover(logger))
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger(logger))
	r.Use(metricsMiddleware())
	r.Use(middleware.CORS(cfg.AllowedOrigins))

	// System endpoints (no auth, no tenant scope).
	r.GET("/livez", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "live"})
	})
	r.GET("/readyz", func(c *gin.Context) {
		rep := healthReg.Report(c.Request.Context())
		status := http.StatusOK
		if !rep.Ready {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, gin.H{"ready": rep.Ready, "live": rep.Live})
	})
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, healthReg.Report(c.Request.Context()))
	})
	r.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"version": Version})
	})
	r.GET("/metrics", gin.WrapH(promhttp.HandlerFor(metrics.Registry(), promhttp.HandlerOpts{})))

	return r
}

// metricsMiddleware observes request latency into the Prometheus histogram
// served at /metrics. It labels by the matched route template (low cardinality)
// — unmatched paths bucket under "unmatched" so a flood of bad URLs can't blow
// up series cardinality.
func metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		metrics.HTTPDuration.
			WithLabelValues(route, c.Request.Method, strconv.Itoa(c.Writer.Status())).
			Observe(time.Since(start).Seconds())
	}
}

// Run wraps http.Server with graceful shutdown.
func Run(ctx context.Context, addr string, h http.Handler, logger *zap.Logger) error {
	srv := &http.Server{Addr: addr, Handler: h}
	errCh := make(chan error, 1)
	go func() {
		logger.Info("http server listening", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutCtx)
	}
}
