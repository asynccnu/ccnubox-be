package metricsx

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewWithRegistererReusesAlreadyRegisteredCollectors(t *testing.T) {
	// 使用独立 registry, 避免污染 prometheus.DefaultRegisterer 全局状态
	registry := prometheus.NewRegistry()

	first := NewWithRegisterer(registry, "ccnubox_test")
	second := NewWithRegisterer(registry, "ccnubox_test")

	if first.HTTP.RequestsTotal != second.HTTP.RequestsTotal {
		t.Fatal("expected HTTP request counter to reuse the registered collector")
	}
	if first.Redis.Duration != second.Redis.Duration {
		t.Fatal("expected Redis duration histogram to reuse the registered collector")
	}
	if first.MQMetrics.FailedTotal != second.MQMetrics.FailedTotal {
		t.Fatal("expected MQ failed counter to reuse the registered collector")
	}
}

func TestNewUsesDefaultRegisterer(t *testing.T) {
	// New 应该走 prometheus.DefaultRegisterer, 校验命名空间前缀
	m := New("ccnubox_default_test")
	defer prometheus.DefaultRegisterer.Unregister(m.HTTP.RequestsTotal)
	defer prometheus.DefaultRegisterer.Unregister(m.Redis.Duration)
	defer prometheus.DefaultRegisterer.Unregister(m.MQMetrics.FailedTotal)

	if m.HTTP.RequestsTotal == nil {
		t.Fatal("expected HTTP requests total to be initialized")
	}
	if m.Redis.Duration == nil {
		t.Fatal("expected Redis duration to be initialized")
	}
	if m.MQMetrics.FailedTotal == nil {
		t.Fatal("expected MQ failed total to be initialized")
	}
}
