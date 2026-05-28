package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/leo/iop/server/internal/shared/kernel"
)

const headerRequestID = "X-Request-Id"

// RequestID echoes a client-provided X-Request-Id or generates a UUID v7 if missing.
// The id is attached to the request context (kernel.TraceIDFromContext) and the
// response headers. Downstream services and logs should propagate it verbatim.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		tid := c.GetHeader(headerRequestID)
		if tid == "" {
			tid = string(kernel.NewID())
		}
		c.Header(headerRequestID, tid)
		ctx := kernel.WithTraceID(c.Request.Context(), tid)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
