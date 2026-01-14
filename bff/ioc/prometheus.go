package ioc

import (
	"github.com/asynccnu/ccnubox-be/bff/conf"
	"github.com/asynccnu/ccnubox-be/common/pkg/prometheusx"
	"github.com/prometheus/client_golang/prometheus"
)

// 感觉划分上不是特别的优雅,但是暂时没更好的办法
func InitPrometheus(cfg *conf.ServerConf) *prometheusx.PrometheusCounter {
	p := prometheusx.NewPrometheus(cfg.Prometheus.Namespace)
	return &prometheusx.PrometheusCounter{
		RouterCounter:     p.RegisterCounter(cfg.Prometheus.RouterCounter.Name, cfg.Prometheus.RouterCounter.Help, []string{"method", "endpoint", "status"}),
		ActiveConnections: p.RegisterGauge(cfg.Prometheus.ActiveConnections.Name, cfg.Prometheus.ActiveConnections.Help, []string{"endpoint"}),
		DurationTime:      p.RegisterHistogram(cfg.Prometheus.DurationTime.Name, cfg.Prometheus.DurationTime.Help, []string{"endpoint", "status"}, prometheus.DefBuckets),
		DailyActiveUsers:  p.RegisterGauge(cfg.Prometheus.DailyActiveUsers.Name, cfg.Prometheus.DailyActiveUsers.Help, []string{"service"}),
	}
}
