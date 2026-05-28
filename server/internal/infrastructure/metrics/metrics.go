package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Registry returns a *prometheus.Registry seeded with default collectors.
// app.go wires this into the Gin /metrics handler.
func Registry() *prometheus.Registry {
	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector())
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	return reg
}

// HTTPDuration is the histogram for request latency by route + status.
var HTTPDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "iop_http_request_duration_seconds",
		Help:    "HTTP request latency.",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"route", "method", "status"},
)
