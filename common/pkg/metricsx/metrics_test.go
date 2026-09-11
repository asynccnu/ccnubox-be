package metricsx

import (
	"strings"
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
	if first.Client.AppErrorsTotal != second.Client.AppErrorsTotal {
		t.Fatal("expected client app error counter to reuse the registered collector")
	}
	if first.Library.PreferenceSyncTotal != second.Library.PreferenceSyncTotal {
		t.Fatal("expected library preference counter to reuse the registered collector")
	}
	if first.Feed.LibraryPublishTotal != second.Feed.LibraryPublishTotal {
		t.Fatal("expected Feed library publish counter to reuse the registered collector")
	}
}

func TestNewUsesDefaultRegisterer(t *testing.T) {
	// New 应该走 prometheus.DefaultRegisterer, 校验命名空间前缀
	m := New("ccnubox_default_test")
	defer prometheus.DefaultRegisterer.Unregister(m.HTTP.RequestsTotal)
	defer prometheus.DefaultRegisterer.Unregister(m.Redis.Duration)
	defer prometheus.DefaultRegisterer.Unregister(m.MQMetrics.FailedTotal)
	defer prometheus.DefaultRegisterer.Unregister(m.Client.AppErrorsTotal)
	defer prometheus.DefaultRegisterer.Unregister(m.Client.APIFailuresTotal)
	defer prometheus.DefaultRegisterer.Unregister(m.Client.StartupDuration)
	defer prometheus.DefaultRegisterer.Unregister(m.Client.IngestedEventsTotal)
	defer prometheus.DefaultRegisterer.Unregister(m.Client.RejectedBatches)
	defer prometheus.DefaultRegisterer.Unregister(m.Library.PreferenceSyncTotal)
	defer prometheus.DefaultRegisterer.Unregister(m.Library.PreferenceSyncLagSeconds)
	defer prometheus.DefaultRegisterer.Unregister(m.Library.RefreshUsersTotal)
	defer prometheus.DefaultRegisterer.Unregister(m.Library.UpstreamRequestsTotal)
	defer prometheus.DefaultRegisterer.Unregister(m.Library.UpstreamDurationSeconds)
	defer prometheus.DefaultRegisterer.Unregister(m.Library.ActiveReservations)
	defer prometheus.DefaultRegisterer.Unregister(m.Library.NotificationJobs)
	defer prometheus.DefaultRegisterer.Unregister(m.Library.NotificationJobLagSeconds)
	defer prometheus.DefaultRegisterer.Unregister(m.Library.Outbox)
	defer prometheus.DefaultRegisterer.Unregister(m.Library.OutboxOldestAgeSeconds)
	defer prometheus.DefaultRegisterer.Unregister(m.Library.NotificationDeduplicatedTotal)
	defer prometheus.DefaultRegisterer.Unregister(m.Library.UnknownReservationStatusTotal)
	defer prometheus.DefaultRegisterer.Unregister(m.Feed.LibraryPublishTotal)
	defer prometheus.DefaultRegisterer.Unregister(m.Feed.PushDeliveryTotal)

	if m.HTTP.RequestsTotal == nil {
		t.Fatal("expected HTTP requests total to be initialized")
	}
	if m.Redis.Duration == nil {
		t.Fatal("expected Redis duration to be initialized")
	}
	if m.MQMetrics.FailedTotal == nil {
		t.Fatal("expected MQ failed total to be initialized")
	}
	if m.Library == nil || m.Library.UpstreamRequestsTotal == nil {
		t.Fatal("expected Library reminder metrics to be initialized")
	}
	if m.Feed == nil || m.Feed.LibraryPublishTotal == nil {
		t.Fatal("expected Feed delivery metrics to be initialized")
	}
}

func TestNewWithRegistererInitializesClientMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := NewWithRegisterer(registry, "ccnubox_test")

	if m.Client == nil || m.Client.AppErrorsTotal == nil || m.Client.StartupDuration == nil {
		t.Fatal("expected client metrics to be initialized")
	}
	// Vec collectors are emitted only after a label set is initialized.
	m.Client.AppErrorsTotal.WithLabelValues("ios", "1.0.0", "course", "error").Inc()
	m.Client.StartupDuration.WithLabelValues("ios", "1.0.0").Observe(1)
	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}
	names := make(map[string]bool, len(families))
	for _, family := range families {
		names[family.GetName()] = true
	}
	if !names["ccnubox_test_app_error_total"] {
		t.Fatal("expected app error metric family")
	}
	if !names["ccnubox_test_app_startup_duration_seconds"] {
		t.Fatal("expected startup duration metric family")
	}
}

func TestNewWithRegistererInitializesUserMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := NewWithRegisterer(registry, "ccnubox_test")

	if m.User == nil {
		t.Fatal("expected User metrics to be initialized")
	}
	if m.User.ActiveUsers24h == nil {
		t.Fatal("expected User.ActiveUsers24h gauge to be initialized")
	}
	got := m.User.ActiveUsers24h.Desc().String()
	if !strings.Contains(got, "ccnubox_test_active_users_24h") {
		t.Fatalf("expected desc to contain 'ccnubox_test_active_users_24h', got: %s", got)
	}
}
