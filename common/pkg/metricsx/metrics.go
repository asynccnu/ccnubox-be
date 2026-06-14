package metricsx

import (
	"errors"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics 所有监控指标的顶层组合
type Metrics struct {
	HTTP      *HTTPMetrics
	Redis     *RedisMetrics
	MQMetrics *MQMetrics
}

// New 创建并初始化所有监控指标，自动注册到 Prometheus 默认 registerer。
// 测试或自定义 registerer 场景请使用 NewWithRegisterer。
func New(namespace string) *Metrics {
	return NewWithRegisterer(prometheus.DefaultRegisterer, namespace)
}

// NewWithRegisterer 把 collector 注册到指定的 registerer(用于测试或自定义 registry)。
// 已注册则复用已存在的实例, 避免重复注册 panic。
func NewWithRegisterer(reg prometheus.Registerer, namespace string) *Metrics {
	m := &Metrics{
		HTTP:      newHTTPMetrics(namespace),
		Redis:     newRedisMetrics(namespace),
		MQMetrics: newMQMetrics(namespace),
	}

	m.HTTP.RequestsTotal = registerVec(reg, m.HTTP.RequestsTotal)
	m.HTTP.Duration = registerVec(reg, m.HTTP.Duration)
	m.HTTP.ActiveConnections = registerVec(reg, m.HTTP.ActiveConnections)
	m.Redis.RequestsTotal = registerVec(reg, m.Redis.RequestsTotal)
	m.Redis.ErrorsTotal = registerVec(reg, m.Redis.ErrorsTotal)
	m.Redis.Duration = registerVec(reg, m.Redis.Duration)
	m.MQMetrics.ProducedTotal = registerVec(reg, m.MQMetrics.ProducedTotal)
	m.MQMetrics.ConsumedTotal = registerVec(reg, m.MQMetrics.ConsumedTotal)
	m.MQMetrics.FailedTotal = registerVec(reg, m.MQMetrics.FailedTotal)

	return m
}

// registerVec 把 collector 注册到给定的 registerer, 已注册则返回已存在的实例(避免重复注册 panic)。
// 类型参数 T 必须实现 prometheus.Collector, 用于在编译期保留具体类型, 避免调用方重复断言。
func registerVec[T prometheus.Collector](reg prometheus.Registerer, c T) T {
	if err := reg.Register(c); err != nil {
		var alreadyRegistered prometheus.AlreadyRegisteredError
		if errors.As(err, &alreadyRegistered) {
			if existing, ok := alreadyRegistered.ExistingCollector.(T); ok {
				return existing
			}
		}
		panic(err)
	}
	return c
}
