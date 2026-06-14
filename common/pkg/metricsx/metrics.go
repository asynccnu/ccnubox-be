package metricsx

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics 所有监控指标的顶层组合
type Metrics struct {
	HTTP      *HTTPMetrics
	Redis     *RedisMetrics
	MQMetrics *MQMetrics
}

// MQ 返回 MQ 指标，供 wire 绑定使用
func (m *Metrics) MQ() *MQMetrics {
	return m.MQMetrics
}

// New 创建并初始化所有监控指标，自动注册到 Prometheus
func New(namespace string) *Metrics {
	m := &Metrics{
		HTTP:      newHTTPMetrics(namespace),
		Redis:     newRedisMetrics(namespace),
		MQMetrics: newMQMetrics(namespace),
	}

	prometheus.MustRegister(
		m.HTTP.RequestsTotal,
		m.HTTP.Duration,
		m.HTTP.ActiveConnections,
		m.Redis.RequestsTotal,
		m.Redis.ErrorsTotal,
		m.Redis.Duration,
		m.MQMetrics.ProducedTotal,
		m.MQMetrics.ConsumedTotal,
		m.MQMetrics.FailedTotal,
		m.MQMetrics.RetryTotal,
	)

	return m
}
