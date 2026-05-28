package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leo/iop/server/internal/shared/kernel"
	"go.uber.org/zap"
)

// Logger logs each request with trace + tenant + member + duration.
// Skips /livez, /healthz, /metrics to keep log volume low.
func Logger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/livez" || path == "/healthz" || path == "/metrics" {
			c.Next()
			return
		}
		start := time.Now()
		c.Next()
		dur := time.Since(start)

		ctx := c.Request.Context()
		tid := kernel.TraceIDFromContext(ctx)
		fields := []zap.Field{
			zap.String("trace_id", tid),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", c.Writer.Status()),
			zap.Int64("duration_ms", dur.Milliseconds()),
			zap.String("remote", c.ClientIP()),
		}
		if tenantID, ok := kernel.TenantIDFromContext(ctx); ok {
			fields = append(fields, zap.String("tenant_id", string(tenantID)))
		}
		if memberID, ok := kernel.MemberIDFromContext(ctx); ok {
			fields = append(fields, zap.String("member_id", string(memberID)))
		}
		if len(c.Errors) > 0 {
			fields = append(fields, zap.Strings("errors", c.Errors.Errors()))
		}
		switch {
		case c.Writer.Status() >= 500:
			logger.Error("http", fields...)
		case c.Writer.Status() >= 400:
			logger.Warn("http", fields...)
		default:
			logger.Info("http", fields...)
		}
	}
}
