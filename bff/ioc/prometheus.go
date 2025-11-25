package ioc

import (
	"github.com/asynccnu/ccnubox-be/bff/pkg/prometheusx"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
)

// 感觉划分上不是特别的优雅,但是暂时没更好的办法
func InitPrometheus() *prometheusx.PrometheusCounter {
	type PrometheusConfig struct {
		Namespace string `yaml:"namespace"` //项目名称

		RouterCounter struct {
			Name string `yaml:"name"`
			Help string `yaml:"help"`
		} `yaml:"routerCounter"`

		ActiveConnections struct {
			Name string `yaml:"name"`
			Help string `yaml:"help"`
		} `yaml:"activeConnections"`

		DurationTime struct {
			Name string `yaml:"name"`
			Help string `yaml:"help"`
		} `yaml:"durationTime"`
	}

	var conf PrometheusConfig
	// 解析配置文件获取 banner 的位置
	err := viper.UnmarshalKey("prometheus", &conf)
	if err != nil {
		panic(err)
	}

	p := prometheusx.NewPrometheus(conf.Namespace)
	return &prometheusx.PrometheusCounter{
		RouterCounter:     p.RegisterCounter(conf.RouterCounter.Name, conf.RouterCounter.Help, []string{"method", "endpoint", "status"}),
		ActiveConnections: p.RegisterGauge(conf.ActiveConnections.Name, conf.RouterCounter.Help, []string{"endpoint"}),
		DurationTime:      p.RegisterHistogram(conf.DurationTime.Name, conf.DurationTime.Help, []string{"endpoint", "status"}, prometheus.DefBuckets),
	}
}
