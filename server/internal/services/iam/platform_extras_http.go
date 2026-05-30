package iam

import (
	"context"
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/services/audit"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
)

// MonitorProbe abstracts the infra health snapshot the monitor route exposes,
// so this package needn't import the health/redis packages. app.go adapts
// health.Registry + the redis client to this interface.
type MonitorProbe interface {
	// Health returns a name→ok/error/latency map for the registered checks.
	Health(ctx context.Context) map[string]any
	// RedisInfo returns a best-effort INFO summary (nil/empty when unavailable).
	RedisInfo(ctx context.Context) map[string]any
}

// RegisterPlatformExtrasRoutes mounts the P3 platform-console surfaces on the
// platform group (caller already applied PlatformAccess + PasswordChangeGate).
// Every mutating route is individually gated with PlatformAuthz(resource:action);
// list/GET routes are gated with the matching :read permission.
//
// probe may be nil (monitor route then reports only db-pool + counters).
func RegisterPlatformExtrasRoutes(r *gin.RouterGroup, svc *Service, aud *audit.Service, pool *pgxpool.Pool, probe MonitorProbe) {
	registerPlatformNoticeRoutes(r, svc, aud, pool)
	registerPlatformParamRoutes(r, svc, aud, pool)
	registerPlatformLogRoutes(r, svc, aud, pool)
	registerPlatformOnlineRoutes(r, svc, aud, pool)
	registerPlatformMonitorRoutes(r, svc, aud, pool, probe)
	registerPlatformCronRoutes(r, svc, aud, pool)
}

func actorID(c *gin.Context) kernel.ID {
	claims, _ := ClaimsFromContext(c.Request.Context())
	return claims.PlatformUserID
}

// ---------------------------------------------------------------- Notices

func registerPlatformNoticeRoutes(r *gin.RouterGroup, svc *Service, aud *audit.Service, pool *pgxpool.Pool) {
	r.GET("/platform/notices", PlatformAuthz(svc, aud, "notice", "read"), func(c *gin.Context) {
		var p kernel.Pagination
		_ = c.ShouldBindQuery(&p)
		items, err := ListPlatformNotices(c.Request.Context(), pool, c.Query("status"), p)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"notices": items})
	})

	r.GET("/platform/notices/:id", PlatformAuthz(svc, aud, "notice", "read"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_id", "id 无效", err))
			return
		}
		n, err := GetPlatformNotice(c.Request.Context(), pool, id)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		if n == nil {
			apiresp.Fail(c, errors.New(errors.KindNotFound, "iam.notice_not_found", "通知不存在"))
			return
		}
		apiresp.OK(c, n)
	})

	r.POST("/platform/notices", PlatformAuthz(svc, aud, "notice", "manage"), func(c *gin.Context) {
		var req struct {
			Title   string `json:"title" binding:"required"`
			Content string `json:"content"`
			Type    string `json:"type"`
			Status  string `json:"status"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_request", "请求格式错误", err))
			return
		}
		n, err := CreatePlatformNotice(c.Request.Context(), pool, req.Title, req.Content, req.Type, req.Status, actorID(c))
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.Created(c, n)
	})

	r.PATCH("/platform/notices/:id", PlatformAuthz(svc, aud, "notice", "manage"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_id", "id 无效", err))
			return
		}
		var req struct {
			Title   string `json:"title" binding:"required"`
			Content string `json:"content"`
			Type    string `json:"type"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_request", "请求格式错误", err))
			return
		}
		if req.Type == "" {
			req.Type = "notice"
		}
		if err := UpdatePlatformNotice(c.Request.Context(), pool, id, req.Title, req.Content, req.Type); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.POST("/platform/notices/:id/publish", PlatformAuthz(svc, aud, "notice", "manage"), func(c *gin.Context) {
		setNoticeStatus(c, pool, "published")
	})
	r.POST("/platform/notices/:id/withdraw", PlatformAuthz(svc, aud, "notice", "manage"), func(c *gin.Context) {
		setNoticeStatus(c, pool, "withdrawn")
	})

	r.DELETE("/platform/notices/:id", PlatformAuthz(svc, aud, "notice", "manage"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_id", "id 无效", err))
			return
		}
		if err := DeletePlatformNotice(c.Request.Context(), pool, id); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})
}

func setNoticeStatus(c *gin.Context, pool *pgxpool.Pool, status string) {
	id, err := kernel.ParseID(c.Param("id"))
	if err != nil {
		apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_id", "id 无效", err))
		return
	}
	if err := SetPlatformNoticeStatus(c.Request.Context(), pool, id, status); err != nil {
		apiresp.Fail(c, err)
		return
	}
	apiresp.OK(c, gin.H{"ok": true})
}

// ---------------------------------------------------------------- Params

func registerPlatformParamRoutes(r *gin.RouterGroup, svc *Service, aud *audit.Service, pool *pgxpool.Pool) {
	r.GET("/platform/params", PlatformAuthz(svc, aud, "param", "read"), func(c *gin.Context) {
		items, err := ListPlatformParams(c.Request.Context(), pool)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"params": items})
	})

	r.PUT("/platform/params/:key", PlatformAuthz(svc, aud, "param", "manage"), func(c *gin.Context) {
		key := c.Param("key")
		if key == "" {
			apiresp.Fail(c, errors.New(errors.KindParam, "iam.invalid_key", "key 必填"))
			return
		}
		var req struct {
			Value json.RawMessage `json:"value"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_request", "请求格式错误", err))
			return
		}
		if len(req.Value) == 0 {
			req.Value = json.RawMessage("null")
		}
		if !json.Valid(req.Value) {
			apiresp.Fail(c, errors.New(errors.KindParam, "iam.invalid_value", "value 必须是合法 JSON"))
			return
		}
		if err := UpsertPlatformParam(c.Request.Context(), pool, key, req.Value, actorID(c)); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.DELETE("/platform/params/:key", PlatformAuthz(svc, aud, "param", "manage"), func(c *gin.Context) {
		key := c.Param("key")
		if err := DeletePlatformParam(c.Request.Context(), pool, key); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})
}

// ---------------------------------------------------------------- Logs

func registerPlatformLogRoutes(r *gin.RouterGroup, svc *Service, aud *audit.Service, pool *pgxpool.Pool) {
	// Operation log: full platform_audit_log with optional actor/action/time filters.
	r.GET("/platform/operlogs", PlatformAuthz(svc, aud, "audit", "read"), func(c *gin.Context) {
		var p kernel.Pagination
		_ = c.ShouldBindQuery(&p)
		f := audit.ListFilter{
			Actor:  c.Query("actor"),
			Action: c.Query("action"),
			From:   c.Query("from"),
			To:     c.Query("to"),
		}
		items, err := aud.ListPlatformFiltered(c.Request.Context(), p, f)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"logs": items})
	})

	// Login log: best-effort. Dedicated login events (iam.user_logged_in/out,
	// iam.login_failed) are published on the in-proc bus and persisted to the
	// *tenant* audit_log, NOT to public.platform_audit_log. So at the platform
	// level we surface whatever login-ish rows exist in platform_audit_log by
	// matching action ILIKE '%login%'. This is currently typically empty —
	// platform-scoped login auditing is a documented limitation/follow-up.
	r.GET("/platform/loginlogs", PlatformAuthz(svc, aud, "audit", "read"), func(c *gin.Context) {
		var p kernel.Pagination
		_ = c.ShouldBindQuery(&p)
		f := audit.ListFilter{
			ActionLike: "%login%",
			Actor:      c.Query("actor"),
			From:       c.Query("from"),
			To:         c.Query("to"),
		}
		items, err := aud.ListPlatformFiltered(c.Request.Context(), p, f)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"logs": items, "note": "platform login events are not yet persisted to platform_audit_log; see code comment"})
	})
}

// ---------------------------------------------------------------- Online users

func registerPlatformOnlineRoutes(r *gin.RouterGroup, svc *Service, aud *audit.Service, pool *pgxpool.Pool) {
	r.GET("/platform/online", PlatformAuthz(svc, aud, "session", "read"), func(c *gin.Context) {
		items, err := ListAllOnlineSessions(c.Request.Context(), pool)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"sessions": items})
	})

	// Kick (force-logout) any session across tenants. Platform-level: no tenant
	// scoping check (PlatformAuthz session:revoke is the gate).
	r.POST("/platform/online/:sid/kick", PlatformAuthz(svc, aud, "session", "revoke"), func(c *gin.Context) {
		sid, err := kernel.ParseID(c.Param("sid"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_session_id", "会话 ID 无效", err))
			return
		}
		if err := svc.Logout(c.Request.Context(), sid); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})
}

// ---------------------------------------------------------------- Monitor

func registerPlatformMonitorRoutes(r *gin.RouterGroup, svc *Service, aud *audit.Service, pool *pgxpool.Pool, probe MonitorProbe) {
	r.GET("/platform/monitor", PlatformAuthz(svc, aud, "monitor", "read"), func(c *gin.Context) {
		ctx := c.Request.Context()
		resp := gin.H{
			"db_pool":  MonitorDBStats(pool),
			"counters": MonitorCounters(ctx, pool),
		}
		if probe != nil {
			resp["health"] = probe.Health(ctx)
			if info := probe.RedisInfo(ctx); len(info) > 0 {
				resp["redis"] = info
			}
		}
		apiresp.OK(c, resp)
	})
}

// ---------------------------------------------------------------- Cron jobs

func registerPlatformCronRoutes(r *gin.RouterGroup, svc *Service, aud *audit.Service, pool *pgxpool.Pool) {
	r.GET("/platform/jobs", PlatformAuthz(svc, aud, "job", "read"), func(c *gin.Context) {
		items, err := ListPlatformJobs(c.Request.Context(), pool)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"jobs": items})
	})

	r.POST("/platform/jobs", PlatformAuthz(svc, aud, "job", "manage"), func(c *gin.Context) {
		var req struct {
			Name     string `json:"name" binding:"required"`
			CronExpr string `json:"cron_expr"`
			Handler  string `json:"handler"`
			Status   string `json:"status"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_request", "请求格式错误", err))
			return
		}
		j, err := CreatePlatformJob(c.Request.Context(), pool, req.Name, req.CronExpr, req.Handler, req.Status)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.Created(c, j)
	})

	r.PATCH("/platform/jobs/:id", PlatformAuthz(svc, aud, "job", "manage"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_id", "id 无效", err))
			return
		}
		cur, err := GetPlatformJob(c.Request.Context(), pool, id)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		if cur == nil {
			apiresp.Fail(c, errors.New(errors.KindNotFound, "iam.job_not_found", "任务不存在"))
			return
		}
		// Pointer fields → partial update (absent = keep current).
		var req struct {
			Name     *string `json:"name"`
			CronExpr *string `json:"cron_expr"`
			Handler  *string `json:"handler"`
			Status   *string `json:"status"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_request", "请求格式错误", err))
			return
		}
		name, cron, handler, status := cur.Name, cur.CronExpr, cur.Handler, cur.Status
		if req.Name != nil {
			name = *req.Name
		}
		if req.CronExpr != nil {
			cron = *req.CronExpr
		}
		if req.Handler != nil {
			handler = *req.Handler
		}
		if req.Status != nil {
			status = *req.Status
		}
		if err := UpdatePlatformJob(c.Request.Context(), pool, id, name, cron, handler, status); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.DELETE("/platform/jobs/:id", PlatformAuthz(svc, aud, "job", "manage"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_id", "id 无效", err))
			return
		}
		if err := DeletePlatformJob(c.Request.Context(), pool, id); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.POST("/platform/jobs/:id/run", PlatformAuthz(svc, aud, "job", "manage"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_id", "id 无效", err))
			return
		}
		run, err := RunPlatformJobNow(c.Request.Context(), pool, id)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"run": run})
	})

	r.GET("/platform/jobs/:id/runs", PlatformAuthz(svc, aud, "job", "read"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_id", "id 无效", err))
			return
		}
		var p kernel.Pagination
		_ = c.ShouldBindQuery(&p)
		runs, err := ListPlatformJobRuns(c.Request.Context(), pool, id, p)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"runs": runs})
	})
}
