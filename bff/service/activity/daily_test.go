package activity

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestDailyMembershipUsesChinaDayAndFixedRetention(t *testing.T) {
	mr := miniredis.RunT(t)
	r := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = r.Close() })
	ctx := context.Background()
	// 16:00 UTC is midnight on the following day in Shanghai.
	now := time.Date(2026, 10, 18, 16, 0, 0, 0, time.UTC)
	mr.SetTime(now)
	for range 3 {
		if err := Record(ctx, r, "a", now); err != nil {
			t.Fatal(err)
		}
	}
	if err := Record(ctx, r, "b", now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	key := "activity:daily:2026-10-19"
	ids, err := r.SMembers(ctx, key).Result()
	if err != nil || len(ids) != 3 {
		t.Fatalf("members = %v, error = %v", ids, err)
	}
	if r.Exists(ctx, "activity:daily:2026-10-18").Val() != 0 {
		t.Fatal("used host/UTC date")
	}
	if ttl := mr.TTL(key); ttl != RetentionDays*24*time.Hour {
		t.Fatalf("TTL = %v", ttl)
	}
	if got, err := r.Get(ctx, StartedKey).Int64(); err != nil || got != now.Unix() {
		t.Fatalf("collection start = %d, error = %v", got, err)
	}
	if err := CheckWindow(ctx, r, Day(now), 1); !errors.Is(err, ErrNotReady) {
		t.Fatalf("first partial day: %v", err)
	}
}

func TestHistoryReadinessReportsStartAndEarliestUsableDate(t *testing.T) {
	mr := miniredis.RunT(t)
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, China)
	mr.SetTime(now)
	r := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer r.Close()
	ctx := context.Background()
	start := Day(now).AddDate(0, 0, -7)
	err := CheckWindow(ctx, r, start, 7)
	var details *HistoryNotReadyError
	if !errors.Is(err, ErrNotReady) || !errors.As(err, &details) || details.Reason != "not_started" || details.WindowStart != "2026-10-13" || details.WindowEnd != "2026-10-19" {
		t.Fatalf("no history: %+v, %v", details, err)
	}
	if !strings.Contains(err.Error(), "Prometheus") {
		t.Fatal("no explanation of historical count limitation")
	}
	since := time.Date(2026, 10, 19, 9, 30, 0, 0, China)
	if err := EnsureDay(ctx, r, since); err != nil {
		t.Fatal(err)
	}
	if err := EnsureDay(ctx, r, now); err != nil {
		t.Fatal(err)
	}
	actual, err := r.Get(ctx, StartedKey).Int64()
	if err != nil || actual != since.Unix() {
		t.Fatalf("collection start changed: %d %v", actual, err)
	}
	err = CheckWindow(ctx, r, start, 7)
	details = nil
	if !errors.As(err, &details) || details.Reason != "collecting" || details.CollectionStartedAt != since.Unix() || details.FirstCompleteDay != "2026-10-20" || details.EarliestReadyDate != "2026-10-27" {
		t.Fatalf("warmup: %+v, %v", details, err)
	}
	if !strings.Contains(err.Error(), "2026-10-27") {
		t.Fatal("earliest ready date omitted")
	}
}
func TestHistoryReadinessIdentifiesMissingDaysAndAcceptsKnownEmptyDays(t *testing.T) {
	mr := miniredis.RunT(t)
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, China)
	mr.SetTime(now)
	r := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer r.Close()
	ctx := context.Background()
	start := Day(now).AddDate(0, 0, -7)
	if err := EnsureDay(ctx, r, start.AddDate(0, 0, -2)); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 7; i++ {
		if i != 1 && i != 5 {
			if err := EnsureDay(ctx, r, start.AddDate(0, 0, i)); err != nil {
				t.Fatal(err)
			}
		}
	}
	err := CheckWindow(ctx, r, start, 7)
	var details *HistoryNotReadyError
	if !errors.Is(err, ErrNotReady) || !errors.As(err, &details) || strings.Join(details.MissingDays, ",") != "2026-10-14,2026-10-18" {
		t.Fatalf("missing dates: %+v %v", details, err)
	}
	for _, i := range []int{1, 5} {
		if err := EnsureDay(ctx, r, start.AddDate(0, 0, i)); err != nil {
			t.Fatal(err)
		}
	}
	if err := CheckWindow(ctx, r, start, 7); err != nil {
		t.Fatalf("known zero-user days should be complete: %v", err)
	}
	r.Close()
	if err := CheckWindow(ctx, r, start, 7); err == nil || errors.Is(err, ErrNotReady) {
		t.Fatalf("Redis failure should remain an infrastructure error: %v", err)
	}
}
