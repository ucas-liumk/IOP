package app

import (
	"context"
	"strings"

	redislib "github.com/redis/go-redis/v9"

	"github.com/leo/iop/server/internal/infrastructure/health"
)

// monitorProbe adapts the in-process health.Registry + redis client to the
// iam.MonitorProbe interface consumed by GET /platform/monitor. Keeps the iam
// package free of health/redis imports.
type monitorProbe struct {
	health *health.Registry
	rdb    *redislib.Client
}

// Health returns each registered dependency check's ok/error/latency.
func (m *monitorProbe) Health(ctx context.Context) map[string]any {
	out := map[string]any{}
	if m.health == nil {
		return out
	}
	rep := m.health.Report(ctx)
	out["live"] = rep.Live
	out["ready"] = rep.Ready
	details := map[string]any{}
	for name, st := range rep.Details {
		entry := map[string]any{"ok": st.OK, "latency_ms": st.MS}
		if st.Err != "" {
			entry["error"] = st.Err
		}
		details[name] = entry
	}
	out["checks"] = details
	return out
}

// RedisInfo returns a small subset of redis INFO (best effort; empty when redis
// is unavailable / degraded mode).
func (m *monitorProbe) RedisInfo(ctx context.Context) map[string]any {
	if m.rdb == nil {
		return nil
	}
	res, err := m.rdb.Info(ctx, "server", "memory", "clients").Result()
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	want := map[string]bool{
		"redis_version":           true,
		"uptime_in_seconds":       true,
		"connected_clients":       true,
		"used_memory_human":       true,
		"used_memory_peak_human":  true,
		"mem_fragmentation_ratio": true,
	}
	out := map[string]any{}
	for _, line := range strings.Split(res, "\n") {
		line = strings.TrimSpace(line)
		k, v, ok := strings.Cut(line, ":")
		if ok && want[k] {
			out[k] = v
		}
	}
	return out
}
