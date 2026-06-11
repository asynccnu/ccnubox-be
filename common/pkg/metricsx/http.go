package metricsx

import (
	"github.com/prometheus/client_golang/prometheus"
)

// HTTPMetrics HTTP 监控指标
type HTTPMetrics struct {
	RequestsTotal     *prometheus.CounterVec
	Duration          *prometheus.HistogramVec
	ActiveConnections *prometheus.GaugeVec
}

func newHTTPMetrics(namespace string) *HTTPMetrics {
	return &HTTPMetrics{
		RequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "http_requests_total",
				Help:      "Total HTTP requests",
			},
			[]string{"method", "endpoint", "status"},
		),
		Duration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request duration",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"endpoint", "status"},
		),
		ActiveConnections: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "http_active_connections",
				Help:      "Active HTTP connections",
			},
			[]string{"endpoint"},
		),
	}
}