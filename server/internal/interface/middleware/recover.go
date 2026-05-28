package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/leo/iop/server/internal/shared/kernel"
	"go.uber.org/zap"
)

// Recover catches panics, logs with trace id + stack, returns 500 JSON.
// Always installed BEFORE any business middleware so panics in those are caught too.
func Recover(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				traceID := kernel.TraceIDFromContext(c.Request.Context())
				logger.Error("panic recovered",
					zap.String("trace_id", traceID),
					zap.String("path", c.Request.URL.Path),
					zap.Any("panic", r),
					zap.ByteString("stack", debug.Stack()),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"code": -1,
					"error": gin.H{
						"code":    "internal.panic",
						"message": "服务器内部错误",
						"kind":    "internal",
					},
					"trace_id": traceID,
				})
			}
		}()
		c.Next()
	}
}
