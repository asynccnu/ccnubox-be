package activity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
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
	if n := r.SCard(ctx, pendingKey(now)).Val(); n != 2 {
		t.Fatalf("durable pending users = %d, want 2", n)
	}
	if err := EnsureDay(ctx, r, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	key := Key(now)
	ids, err := r.SMembers(ctx, key).Result()
	if err != nil || len(ids) != 3 {
		t.Fatalf("members = %v, error = %v", ids, err)
	}
	if r.Exists(ctx, Key(now.Add(-time.Second))).Val() != 0 {
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
	if err := EnsureDay(ctx, r, now); err != nil {
		t.Fatal(err)
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
	if err := EnsureDay(ctx, r, now); err != nil {
		t.Fatal(err)
	}
	if err := CheckWindow(ctx, r, start, 7); err != nil {
		t.Fatalf("known zero-user days should be complete: %v", err)
	}
	r.Close()
	if err := CheckWindow(ctx, r, start, 7); err == nil || errors.Is(err, ErrNotReady) {
		t.Fatalf("Redis failure should remain an infrastructure error: %v", err)
	}
}

func TestHistoryRequiresCompletionAfterDurableReplay(t *testing.T) {
	mr := miniredis.RunT(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, China)
	mr.SetTime(now)
	r := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = r.Close() })
	day := Day(now).AddDate(0, 0, -1)
	if err := EnsureDay(ctx, r, day.AddDate(0, 0, -1)); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"written", "pending"} {
		if err := Record(ctx, r, id, day); err != nil {
			t.Fatal(err)
		}
	}
	// One successfully materialized user must not hide another pending write.
	if err := r.SAdd(ctx, Key(day), "written").Err(); err != nil {
		t.Fatal(err)
	}
	if err := r.SRem(ctx, pendingKey(day), "written").Err(); err != nil {
		t.Fatal(err)
	}
	if err := CheckWindow(ctx, r, day, 1); !errors.Is(err, ErrNotReady) {
		t.Fatalf("partially written nonempty day was accepted: %v", err)
	}
	// A restarted worker recovers the intent without the original goroutine.
	r2 := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer r2.Close()
	if err := EnsureDay(ctx, r2, now); err != nil {
		t.Fatal(err)
	}
	if err := CheckWindow(ctx, r2, day, 1); err != nil {
		t.Fatalf("replayed day not complete: %v", err)
	}
	if n := r2.SCard(ctx, Key(day)).Val(); n != 3 {
		t.Fatalf("missing replayed user: size=%d", n)
	}
	if err := Record(ctx, r2, "late", day); !errors.Is(err, ErrDayClosed) {
		t.Fatalf("sealed day accepted a late writer: %v", err)
	}
	if r2.Exists(ctx, CompleteKey(now)).Val() != 0 {
		t.Fatal("current natural day was certified")
	}
	if ttl := mr.TTL(CompleteKey(day)); ttl != mr.TTL(Key(day)) {
		t.Fatalf("completion retention=%v members=%v", ttl, mr.TTL(Key(day)))
	}
}

func TestLostIntentsOrMembersCannotBeCertified(t *testing.T) {
	for _, lost := range []string{"pending", "members"} {
		t.Run(lost, func(t *testing.T) {
			mr := miniredis.RunT(t)
			now := time.Date(2026, 10, 20, 12, 0, 0, 0, China)
			mr.SetTime(now)
			r := redis.NewClient(&redis.Options{Addr: mr.Addr()})
			defer r.Close()
			ctx := context.Background()
			day := Day(now).AddDate(0, 0, -1)
			if err := EnsureDay(ctx, r, day.AddDate(0, 0, -1)); err != nil {
				t.Fatal(err)
			}
			if err := Record(ctx, r, "missing", day); err != nil {
				t.Fatal(err)
			}
			if lost == "pending" {
				r.Del(ctx, pendingKey(day))
			} else {
				r.Del(ctx, Key(day))
			}
			if err := EnsureDay(ctx, r, now); err == nil {
				t.Fatal("lost admitted activity was silently reconstructed")
			}
			if r.Exists(ctx, CompleteKey(day)).Val() != 0 {
				t.Fatal("lost activity was certified")
			}
			if err := CheckWindow(ctx, r, day, 1); !errors.Is(err, ErrNotReady) {
				t.Fatalf("lost activity was selectable: %v", err)
			}
		})
	}
}

func TestReplayFailureDoesNotCreateCompletionAndCanRetry(t *testing.T) {
	mr := miniredis.RunT(t)
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, China)
	mr.SetTime(now)
	r := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer r.Close()
	ctx := context.Background()
	day := Day(now).AddDate(0, 0, -1)
	if err := EnsureDay(ctx, r, day.AddDate(0, 0, -1)); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 600; i++ {
		if err := Record(ctx, r, fmt.Sprintf("user-%d", i), day); err != nil {
			t.Fatal(err)
		}
	}
	// A Redis runtime failure must leave the pending intents intact.
	r.Del(ctx, Key(day))
	r.Set(ctx, Key(day), "wrong-type", 0)
	if err := EnsureDay(ctx, r, now); err == nil {
		t.Fatal("replay failure ignored")
	}
	if r.SCard(ctx, pendingKey(day)).Val() != 600 || r.Exists(ctx, CompleteKey(day)).Val() != 0 {
		t.Fatal("failed replay lost pending users or certified the day")
	}
	// Repair the damaged empty destination, then replay more than one batch.
	r.Del(ctx, Key(day))
	r.SAdd(ctx, Key(day), "")
	if err := EnsureDay(ctx, r, now); err != nil {
		t.Fatal(err)
	}
	if err := CheckWindow(ctx, r, day, 1); err != nil {
		t.Fatal(err)
	}
	if r.SCard(ctx, Key(day)).Val() != 601 {
		t.Fatal("replay failed to drain all batches")
	}
	// Losing a subset after sealing must also invalidate the snapshot.
	r.SRem(ctx, Key(day), "user-1")
	if ids, err := ReadDay(ctx, r, day); err != nil || len(ids) != 0 {
		t.Fatalf("read partially deleted snapshot: ids=%d err=%v", len(ids), err)
	}
	if err := CheckWindow(ctx, r, day, 1); !errors.Is(err, ErrNotReady) {
		t.Fatalf("partially deleted sealed set was accepted: %v", err)
	}
}

func TestClosingDaySerializesWithConcurrentAdmissions(t *testing.T) {
	mr := miniredis.RunT(t)
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, China)
	mr.SetTime(now)
	r := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer r.Close()
	ctx := context.Background()
	day := Day(now).AddDate(0, 0, -1)
	if err := EnsureDay(ctx, r, day.AddDate(0, 0, -1)); err != nil {
		t.Fatal(err)
	}
	if err := Record(ctx, r, "before-close", day); err != nil {
		t.Fatal(err)
	}
	accepted := make(chan string, 50)
	failures := make(chan error, 50)
	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id := fmt.Sprintf("concurrent-%d", i)
			err := Record(ctx, r, id, day)
			if err == nil {
				accepted <- id
			} else if !errors.Is(err, ErrDayClosed) {
				failures <- err
			}
		}()
	}
	if err := EnsureDay(ctx, r, now); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	close(accepted)
	close(failures)
	for err := range failures {
		t.Fatal(err)
	}
	ids, err := ReadDay(ctx, r, day)
	if err != nil {
		t.Fatal(err)
	}
	got := make(map[string]bool, len(ids))
	for _, id := range ids {
		got[id] = true
	}
	want := 2 // admitted before closing, plus the empty-day sentinel
	for id := range accepted {
		want++
		if !got[id] {
			t.Fatalf("sealed day lost admitted user %q", id)
		}
	}
	if !got["before-close"] || len(got) != want {
		t.Fatalf("sealed members=%d want=%d", len(got), want)
	}
}
