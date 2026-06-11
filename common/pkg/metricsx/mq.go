package metricsx

import (
	"github.com/prometheus/client_golang/prometheus"
)

// MQMetrics MQ 监控指标
type MQMetrics struct {
	ProducedTotal *prometheus.CounterVec
	ConsumedTotal *prometheus.CounterVec
	FailedTotal   *prometheus.CounterVec
	RetryTotal    *prometheus.CounterVec
}

func newMQMetrics(namespace string) *MQMetrics {
	return &MQMetrics{
		ProducedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "mq_produced_total",
				Help:      "Total messages produced",
			},
			[]string{"topic", "status"},
		),
		ConsumedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "mq_consumed_total",
				Help:      "Total messages consumed",
			},
			[]string{"topic", "status"},
		),
		FailedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "mq_failed_total",
				Help:      "Total message failures",
			},
			[]string{"topic", "error_type"},
		),
		RetryTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "mq_retry_total",
				Help:      "Total message retries",
			},
			[]string{"topic"},
		),
	}
}