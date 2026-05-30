package middleware

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/leo/iop/server/internal/shared/kernel"
)

// RateLimitConfig defines per-tenant / per-user / per-IP windows.
type RateLimitConfig struct {
	TenantPerMin int
	UserPerMin   int
	IPPerMin     int
}

func DefaultRateLimit() RateLimitConfig {
	return RateLimitConfig{TenantPerMin: 1000, UserPerMin: 60, IPPerMin: 20}
}

// DevRateLimit is roomier so a single developer machine (one IP) and E2E suites
// don't trip the anonymous-IP throttle while hammering /auth/login etc.
func DevRateLimit() RateLimitConfig {
	return RateLimitConfig{TenantPerMin: 10000, UserPerMin: 600, IPPerMin: 600}
}

// RateLimit returns a Gin middleware that enforces sliding-window counters via Redis.
// If rdb is nil OR Redis errors, it degrades to an in-memory counter (logs nothing here;
// caller sees 429 response).
func RateLimit(rdb *redis.Client, cfg RateLimitConfig) gin.HandlerFunc {
	mem := newMemoryCounter()
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		now := time.Now().Unix()
		window := int64(60)

		check := func(key string, limit int) (allowed bool, retryAfter int) {
			if limit <= 0 {
				return true, 0
			}
			if rdb != nil {
				k := "rl:" + key + ":" + strconv.FormatInt(now/window, 10)
				count, err := rdb.Incr(ctx, k).Result()
				if err == nil {
					if count == 1 {
						_ = rdb.Expire(ctx, k, time.Duration(window)*time.Second).Err()
					}
					if count > int64(limit) {
						return false, int(window - now%window)
					}
					return true, 0
				}
				// fall through to memory counter on Redis error
			}
			return mem.allow(key, limit, time.Duration(window)*time.Second), int(window - now%window)
		}

		// Tenant
		if tid, ok := kernel.TenantIDFromContext(ctx); ok && tid != "" {
			if ok, ra := check("t:"+string(tid), cfg.TenantPerMin); !ok {
				deny(c, ra, "tenant_per_min")
				return
			}
		}
		// User (member)
		if mid, ok := kernel.MemberIDFromContext(ctx); ok && mid != "" {
			if ok, ra := check("m:"+string(mid), cfg.UserPerMin); !ok {
				deny(c, ra, "user_per_min")
				return
			}
		} else {
			// Anonymous → throttle by IP for public endpoints
			ip := c.ClientIP()
			if ok, ra := check("ip:"+ip, cfg.IPPerMin); !ok {
				deny(c, ra, "ip_per_min")
				return
			}
		}
		c.Next()
	}
}

func deny(c *gin.Context, retryAfter int, scope string) {
	c.Header("Retry-After", strconv.Itoa(retryAfter))
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
		"code":     -1,
		"error":    gin.H{"code": "ratelimit." + scope, "message": "请求过于频繁", "kind": "rate_limit"},
		"trace_id": kernel.TraceIDFromContext(c.Request.Context()),
	})
}

// in-memory fallback counter
type memoryCounter struct {
	mu   sync.Mutex
	data map[string]*memBucket
}

type memBucket struct {
	count   int
	expires time.Time
}

func newMemoryCounter() *memoryCounter {
	return &memoryCounter{data: map[string]*memBucket{}}
}

func (m *memoryCounter) allow(key string, limit int, ttl time.Duration) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	b, ok := m.data[key]
	if !ok || now.After(b.expires) {
		m.data[key] = &memBucket{count: 1, expires: now.Add(ttl)}
		return true
	}
	b.count++
	return b.count <= limit
}

// satisfy ctx unused linter
var _ = context.Background
