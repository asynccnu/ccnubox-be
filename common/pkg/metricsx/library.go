package metricsx

import "github.com/prometheus/client_golang/prometheus"

// LibraryReminderMetrics 包含持久化图书馆提醒流水线所需的运维指标。
// 标签仅使用有限枚举，学号、预约 ID 和去重键不得作为指标标签。
type LibraryReminderMetrics struct {
	PreferenceSyncTotal           *prometheus.CounterVec
	PreferenceSyncLagSeconds      prometheus.Gauge
	RefreshUsersTotal             *prometheus.CounterVec
	UpstreamRequestsTotal         *prometheus.CounterVec
	UpstreamDurationSeconds       *prometheus.HistogramVec
	ActiveReservations            prometheus.Gauge
	NotificationJobs              *prometheus.GaugeVec
	NotificationJobLagSeconds     prometheus.Gauge
	Outbox                        *prometheus.GaugeVec
	OutboxOldestAgeSeconds        prometheus.Gauge
	NotificationDeduplicatedTotal *prometheus.CounterVec
	UnknownReservationStatusTotal prometheus.Counter
}

func newLibraryReminderMetrics(namespace string) *LibraryReminderMetrics {
	return &LibraryReminderMetrics{
		PreferenceSyncTotal:           prometheus.NewCounterVec(prometheus.CounterOpts{Namespace: namespace, Name: "library_preference_sync_total", Help: "Library preference synchronization runs by result."}, []string{"result"}),
		PreferenceSyncLagSeconds:      prometheus.NewGauge(prometheus.GaugeOpts{Namespace: namespace, Name: "library_preference_sync_lag_seconds", Help: "Age of the newest applied Feed preference revision."}),
		RefreshUsersTotal:             prometheus.NewCounterVec(prometheus.CounterOpts{Namespace: namespace, Name: "library_refresh_users_total", Help: "Library user refreshes by result."}, []string{"result"}),
		UpstreamRequestsTotal:         prometheus.NewCounterVec(prometheus.CounterOpts{Namespace: namespace, Name: "library_upstream_requests_total", Help: "School library requests by endpoint and result."}, []string{"endpoint", "result"}),
		UpstreamDurationSeconds:       prometheus.NewHistogramVec(prometheus.HistogramOpts{Namespace: namespace, Name: "library_upstream_duration_seconds", Help: "School library request latency.", Buckets: prometheus.DefBuckets}, []string{"endpoint"}),
		ActiveReservations:            prometheus.NewGauge(prometheus.GaugeOpts{Namespace: namespace, Name: "library_active_reservations", Help: "Users with an active reservation window."}),
		NotificationJobs:              prometheus.NewGaugeVec(prometheus.GaugeOpts{Namespace: namespace, Name: "library_notification_jobs", Help: "Persisted reminder jobs by type and status."}, []string{"type", "status"}),
		NotificationJobLagSeconds:     prometheus.NewGauge(prometheus.GaugeOpts{Namespace: namespace, Name: "library_notification_job_lag_seconds", Help: "Age of the oldest due pending reminder job."}),
		Outbox:                        prometheus.NewGaugeVec(prometheus.GaugeOpts{Namespace: namespace, Name: "library_outbox", Help: "Persisted library outbox rows by status."}, []string{"status"}),
		OutboxOldestAgeSeconds:        prometheus.NewGauge(prometheus.GaugeOpts{Namespace: namespace, Name: "library_outbox_oldest_age_seconds", Help: "Age of the oldest sendable library outbox row."}),
		NotificationDeduplicatedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{Namespace: namespace, Name: "library_notification_deduplicated_total", Help: "Library facts suppressed by a stable dedupe key."}, []string{"type"}),
		UnknownReservationStatusTotal: prometheus.NewCounter(prometheus.CounterOpts{Namespace: namespace, Name: "library_unknown_reservation_status_total", Help: "Reservation rows whose upstream status is not in the validated enum."}),
	}
}

type FeedDeliveryMetrics struct {
	LibraryPublishTotal *prometheus.CounterVec
	PushDeliveryTotal   *prometheus.CounterVec
}

func newFeedDeliveryMetrics(namespace string) *FeedDeliveryMetrics {
	return &FeedDeliveryMetrics{
		LibraryPublishTotal: prometheus.NewCounterVec(prometheus.CounterOpts{Namespace: namespace, Name: "feed_library_publish_total", Help: "Library Feed publish requests by final status."}, []string{"status"}),
		PushDeliveryTotal:   prometheus.NewCounterVec(prometheus.CounterOpts{Namespace: namespace, Name: "feed_push_delivery_total", Help: "Durable Feed push deliveries by result."}, []string{"result"}),
	}
}
