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
	if first.Library.InvitationRequestsTotal != second.Library.InvitationRequestsTotal || first.Library.PreferenceCaughtUpAt != second.Library.PreferenceCaughtUpAt {
		t.Fatal("invitation collectors were not reused")
	}
	if first.Library.PreferenceSyncTotal != second.Library.PreferenceSyncTotal {
		t.Fatal("expected library preference counter to reuse the registered collector")
	}
	if first.Library.TeamScanUsersTotal != second.Library.TeamScanUsersTotal {
		t.Fatal("expected team scan counter to be reused")
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
	defer prometheus.DefaultRegisterer.Unregister(m.Library.InvitationRequestsTotal)
	defer prometheus.DefaultRegisterer.Unregister(m.Library.PreferenceCaughtUpAt)
	defer prometheus.DefaultRegisterer.Unregister(m.Library.PreferenceSyncTotal)
	defer prometheus.DefaultRegisterer.Unregister(m.Library.PreferenceSyncLagSeconds)
	defer prometheus.DefaultRegisterer.Unregister(m.Library.RefreshUsersTotal)
	defer prometheus.DefaultRegisterer.Unregister(m.Library.TeamScanUsersTotal)
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

func TestInvitationMetricRegistration(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewWithRegisterer(reg, "invitation_test")
	m.Library.InvitationRequestsTotal.WithLabelValues("accepted").Inc()
	m.Library.PreferenceCaughtUpAt.Set(123)
	families, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, family := range families {
		switch family.GetName() {
		case "invitation_test_library_invitation_requests_total":
			if len(family.Metric) != 1 || len(family.Metric[0].Label) != 1 || family.Metric[0].Label[0].GetName() != "result" {
				t.Fatal("unexpected labels")
			}
			found++
		case "invitation_test_library_preference_caught_up_timestamp_seconds":
			if family.Metric[0].Gauge.GetValue() != 123 {
				t.Fatal("unexpected timestamp")
			}
			found++
		}
	}
	if found != 2 {
		t.Fatalf("metrics found=%d", found)
	}
}
