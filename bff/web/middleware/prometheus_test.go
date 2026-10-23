package middleware

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/asynccnu/ccnubox-be/bff/cron"
	"github.com/asynccnu/ccnubox-be/bff/service/activity"
	"github.com/asynccnu/ccnubox-be/common/pkg/metricsx"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

func TestPrometheusMiddlewareRecordsCurrentBucket(t *testing.T) {
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
	mr.SetTime(now)

	if err := middleware.recordActiveUser(context.Background(), "stu-1", now); err != nil {
		t.Fatalf("record first active user: %v", err)
	}
	if err := middleware.recordActiveUser(context.Background(), "stu-2", now.Add(time.Minute)); err != nil {
		t.Fatalf("record second active user: %v", err)
	}

	bucketKey := cron.ActiveUsersBucketKey(now)
	count, err := client.PFCount(context.Background(), bucketKey).Result()
	if err != nil {
		t.Fatalf("count bucket: %v", err)
	}
	if count != 2 {
		t.Fatalf("bucket count got %d, want 2", count)
	}
	if n := client.Exists(context.Background(), activity.Key(now)).Val(); n != 0 {
		t.Fatal("best-effort metric path must not collect exact activity")
	}
	if ttl := mr.TTL(bucketKey); ttl <= 24*time.Hour || ttl > cron.ActiveUsersBucketTTL() {
		t.Fatalf("bucket TTL got %s, want in (24h, %s]", ttl, cron.ActiveUsersBucketTTL())
	}
}
