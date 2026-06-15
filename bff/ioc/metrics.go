package ioc

import (
	"github.com/asynccnu/ccnubox-be/bff/conf"
	"github.com/asynccnu/ccnubox-be/bff/web/metrics"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/pkg/metricsx"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

func InitMetrics(cfg *conf.ServerConf) *metricsx.Metrics {
	return metricsx.New("ccnubox")
}

// InitDAUCollector 构造 DAUCollector 并注册到默认 Prometheus registry,
// 让 BFF 的 /metrics 端点自动暴露 ccnubox_dau{date=...} 指标。
func InitDAUCollector(m *metricsx.Metrics, redisClient redis.Cmdable, l logger.Logger) *metrics.DAUCollector {
	collector := metrics.NewDAUCollector(m.User.DAU, redisClient, l)
	prometheus.MustRegister(collector)
	return collector
}
