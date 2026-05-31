package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/leo/iop/server/internal/shared/kernel"
)

const (
	idempotencyHeader = "Idempotency-Key"
	idempotencyTTL    = 24 * time.Hour
)

// Idempotency requires Idempotency-Key on POST/PATCH/DELETE and caches the first response.
// If rdb is nil the middleware is a no-op (M1 dev convenience; M2 prod expects Redis).
func Idempotency(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		if rdb == nil {
			c.Next()
			return
		}
		method := c.Request.Method
		if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
			c.Next()
			return
		}
		key := c.GetHeader(idempotencyHeader)
		if key == "" {
			// Strict mode would 400 here; v3.1 allows missing key (logged via middleware/logger).
			c.Next()
			return
		}
		// Compose a scoped key: tenant + authenticated principal + client key.
		tid, _ := kernel.TenantIDFromContext(c.Request.Context())
		mid, _ := kernel.MemberIDFromContext(c.Request.Context())
		principal := "m:" + string(mid)
		if mid == "" {
			if uid, ok := kernel.PlatformUserIDFromContext(c.Request.Context()); ok && uid != "" {
				principal = "u:" + string(uid)
			} else {
				principal = "ip:" + c.ClientIP()
			}
		}
		rkey := "idem:" + string(tid) + ":" + principal + ":" + key

		ctx := c.Request.Context()
		val, err := rdb.Get(ctx, rkey).Result()
		if err == nil && val != "" {
			// Cache hit: replay response
			var cached cachedResponse
			if json.Unmarshal([]byte(val), &cached) == nil {
				for k, vs := range cached.Headers {
					for _, v := range vs {
						c.Writer.Header().Add(k, v)
					}
				}
				c.Header("X-Idempotency-Replay", "1")
				c.Writer.WriteHeader(cached.Status)
				_, _ = c.Writer.Write(cached.Body)
				c.Abort()
				return
			}
		}
		// Cache miss: capture response.
		w := &captureWriter{ResponseWriter: c.Writer, body: bytes.NewBuffer(nil)}
		c.Writer = w
		c.Next()

		// Only cache 2xx and 4xx (deterministic responses). 5xx allowed to retry.
		status := w.status
		if status >= 200 && status < 500 {
			cached := cachedResponse{
				Status:  status,
				Headers: filterHeaders(w.Header()),
				Body:    w.body.Bytes(),
			}
			raw, _ := json.Marshal(cached)
			_ = rdb.Set(ctx, rkey, raw, idempotencyTTL).Err()
		}
		_ = strconv.Itoa
	}
}

type cachedResponse struct {
	Status  int                 `json:"status"`
	Headers map[string][]string `json:"headers"`
	Body    []byte              `json:"body"`
}

func filterHeaders(h http.Header) map[string][]string {
	out := map[string][]string{}
	for k, v := range h {
		switch k {
		case "Content-Type":
			out[k] = v
		}
	}
	return out
}

type captureWriter struct {
	gin.ResponseWriter
	status int
	body   *bytes.Buffer
}

func (w *captureWriter) WriteHeader(s int) {
	w.status = s
	w.ResponseWriter.WriteHeader(s)
}

func (w *captureWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = 200
	}
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *captureWriter) WriteString(s string) (int, error) {
	if w.status == 0 {
		w.status = 200
	}
	w.body.WriteString(s)
	return io.WriteString(w.ResponseWriter, s)
}
