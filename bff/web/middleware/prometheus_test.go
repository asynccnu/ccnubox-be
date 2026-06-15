package middleware

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/asynccnu/ccnubox-be/bff/cron"
	"github.com/asynccnu/ccnubox-be/common/pkg/metricsx"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/redis/go-redis/v9"
)

func TestPrometheusMiddlewareRecordDAUUpdatesGauge(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	metrics := metricsx.NewWithRegisterer(prometheus.NewRegistry(), "test")
	middleware := NewPrometheusMiddleware(metrics, client)
	now := time.Date(2026, 6, 16, 10, 7, 0, 0, time.Local)

	if err := middleware.recordDAU(context.Background(), "stu-1", now); err != nil {
		t.Fatalf("record first dau: %v", err)
	}
	if err := middleware.recordDAU(context.Background(), "stu-2", now.Add(time.Minute)); err != nil {
		t.Fatalf("record second dau: %v", err)
	}

	if got := testutil.ToFloat64(metrics.User.DAU); got != 2 {
		t.Fatalf("gauge got %v, want 2", got)
	}

	dayCount, err := client.PFCount(context.Background(), cron.DAUDayKeyForTime(now)).Result()
	if err != nil {
		t.Fatalf("count day key: %v", err)
	}
	if dayCount != 2 {
		t.Fatalf("day key count got %d, want 2", dayCount)
	}
}
