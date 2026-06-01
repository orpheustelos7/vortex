package observability

import (
"net/http"
"time"

"github.com/prometheus/client_golang/prometheus"
"github.com/prometheus/client_golang/prometheus/promauto"
"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
Requests    *prometheus.CounterVec
Errors      *prometheus.CounterVec
RateLimited prometheus.Counter
Latency     *prometheus.HistogramVec
}

func NewMetrics() *Metrics {
return &Metrics{
Requests: promauto.NewCounterVec(prometheus.CounterOpts{Name: "vortex_requests_total", Help: "Total requests by tenant and status."}, []string{"tenant", "status"}),
Errors: promauto.NewCounterVec(prometheus.CounterOpts{Name: "vortex_errors_total", Help: "Gateway errors by type."}, []string{"type"}),
RateLimited: promauto.NewCounter(prometheus.CounterOpts{Name: "vortex_rate_limited_total", Help: "Rate limited requests."}),
Latency: promauto.NewHistogramVec(prometheus.HistogramOpts{Name: "vortex_request_latency_seconds", Help: "Gateway request latency.", Buckets: prometheus.DefBuckets}, []string{"tenant"}),
}
}

func (m *Metrics) Handler() http.Handler {
return promhttp.Handler()
}

func (m *Metrics) Observe(tenant string, status int, started time.Time) {
m.Requests.WithLabelValues(tenant, http.StatusText(status)).Inc()
m.Latency.WithLabelValues(tenant).Observe(time.Since(started).Seconds())
}
