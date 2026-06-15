package metricsx

import "github.com/prometheus/client_golang/prometheus"

// UserMetrics 用户行为相关指标
// DAU 是一个无标签 Gauge, 由 cron 任务在每天 00:05 写入"昨天"的最终值
type UserMetrics struct {
	DAU prometheus.Gauge
}

func newUserMetrics(namespace string) *UserMetrics {
	return &UserMetrics{
		DAU: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, "", "dau"),
			Help: "Daily active users (unique StudentId per day, finalized at 00:05 local).",
		}),
	}
}
