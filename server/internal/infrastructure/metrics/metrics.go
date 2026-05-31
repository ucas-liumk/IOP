package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// reg is the SINGLE registry served at /metrics. All IOP custom metrics must
// register here (via Registry()/Registerer()), not the global prometheus
// default registry — otherwise they are created but never scraped. Seeded with
// the Go runtime + process collectors.
var reg = newRegistry()

func newRegistry() *prometheus.Registry {
	r := prometheus.NewRegistry()
	r.MustRegister(collectors.NewGoCollector())
	r.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	return r
}

// Registry returns the shared registry wired into the Gin /metrics handler.
func Registry() *prometheus.Registry { return reg }

// Registerer exposes the served registry so other packages (e.g.
// infrastructure/pg) register their collectors on the SAME registry /metrics
// serves, keeping every custom metric scrapeable.
func Registerer() prometheus.Registerer { return reg }

// HTTPDuration is the request-latency histogram. It is observed by the metrics
// middleware (interface/server.go) and registered on the served registry above
// so iop_http_request_duration_seconds actually appears at /metrics.
var HTTPDuration = func() *prometheus.HistogramVec {
	h := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "iop_http_request_duration_seconds",
			Help:    "HTTP request latency in seconds, by route, method and status.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"route", "method", "status"},
	)
	reg.MustRegister(h)
	return h
}()
