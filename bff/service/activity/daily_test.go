package activity

import (
	"context"
	"errors"
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
