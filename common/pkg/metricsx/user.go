package metricsx

import "github.com/prometheus/client_golang/prometheus"

// UserMetrics 用户行为相关指标 (目前只有 DAU)。
// DAU 是 *prometheus.Desc 定义, 实际 emit 由 DAUCollector 完成,
// 不在 NewWithRegisterer 中注册以避免与 Collector 重复 emit。
type UserMetrics struct {
	DAU *prometheus.Desc
}

func newUserMetrics(namespace string) *UserMetrics {
	return &UserMetrics{
		DAU: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "dau"),
			"Daily active users (unique StudentId per day, rolling 30d window). "+
				"Multi-pod: use avg() or min() in queries, NOT sum() — each pod reads identical Redis data.",
			[]string{"date"},
			nil,
		),
	}
}
