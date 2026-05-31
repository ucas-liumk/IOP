package pg

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"

	"github.com/leo/iop/server/internal/infrastructure/metrics"
)

// SlowQueryMs is the warn threshold; queries over this are logged + counted.
const SlowQueryMs = 200

// Registered on the served metrics registry (not the global default) so that
// iop_pg_slow_query_total is actually scrapeable at /metrics.
var slowQueryTotal = promauto.With(metrics.Registerer()).NewCounterVec(prometheus.CounterOpts{
	Name: "iop_pg_slow_query_total",
	Help: "PostgreSQL queries that exceeded slow query threshold.",
}, []string{"severity"})

// SlowQueryTracer implements pgx.QueryTracer.
type SlowQueryTracer struct {
	logger *zap.Logger
}

func NewSlowQueryTracer(logger *zap.Logger) *SlowQueryTracer {
	return &SlowQueryTracer{logger: logger}
}

type queryStartKey struct{}

type queryCtx struct {
	start time.Time
	sql   string
}

func (t *SlowQueryTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	return context.WithValue(ctx, queryStartKey{}, &queryCtx{start: time.Now(), sql: data.SQL})
}

func (t *SlowQueryTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	qc, ok := ctx.Value(queryStartKey{}).(*queryCtx)
	if !ok {
		return
	}
	dur := time.Since(qc.start)
	ms := dur.Milliseconds()
	if ms < SlowQueryMs {
		return
	}
	severity := "warn"
	if ms >= 1000 {
		severity = "error"
	}
	slowQueryTotal.WithLabelValues(severity).Inc()

	fields := []zap.Field{
		zap.Int64("duration_ms", ms),
		zap.String("sql", truncate(qc.sql, 200)),
	}
	if data.Err != nil {
		fields = append(fields, zap.Error(data.Err))
	}
	if severity == "error" {
		t.logger.Error("slow query (>=1s)", fields...)
	} else {
		t.logger.Warn("slow query (>200ms)", fields...)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
