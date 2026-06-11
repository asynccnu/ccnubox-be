package metricsx

import (
	"github.com/prometheus/client_golang/prometheus"
)

// RedisMetrics Redis 监控指标
type RedisMetrics struct {
	RequestsTotal *prometheus.CounterVec
	ErrorsTotal   *prometheus.CounterVec
	Duration      *prometheus.HistogramVec
}

func newRedisMetrics(namespace string) *RedisMetrics {
	return &RedisMetrics{
		RequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "redis_requests_total",
				Help:      "Total Redis requests",
			},
			[]string{"operation", "status"},
		),
		ErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "redis_errors_total",
				Help:      "Total Redis errors",
			},
			[]string{"operation", "error_type"},
		),
		Duration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "redis_duration_seconds",
				Help:      "Redis operation duration",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"operation"},
		),
	}
}