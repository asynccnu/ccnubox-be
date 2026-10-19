package activity

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	RetentionDays = 32
	StartedKey    = "activity:daily:started_at"
)

// China is independent of the Pod's local timezone and needs no host tzdata.
var China = time.FixedZone("Asia/Shanghai", 8*60*60)

var ErrNotReady = errors.New("活跃历史不足或日期数据缺失，请积累完整自然日数据后重试")

func Day(t time.Time) time.Time {
	t = t.In(China)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, China)
}

func Key(t time.Time) string { return "activity:daily:" + t.In(China).Format("2006-01-02") }

// An empty member marks a known day, including days with zero active users.
// The first (possibly partial) collection day is never eligible for selection.
func Record(ctx context.Context, r redis.Cmdable, studentID string, now time.Time) error {
	_, err := r.TxPipelined(ctx, func(p redis.Pipeliner) error {
		p.SetNX(ctx, StartedKey, now.Unix(), 0)
		p.SAdd(ctx, Key(now), "", studentID)
		p.ExpireAt(ctx, Key(now), Day(now).AddDate(0, 0, RetentionDays))
		return nil
	})
	return err
}

// EnsureDay is called periodically even without traffic. Missing days are not
// silently interpreted as zero activity. Collection is best-effort, like DAU.
func EnsureDay(ctx context.Context, r redis.Cmdable, now time.Time) error {
	return Record(ctx, r, "", now)
}

func CheckWindow(ctx context.Context, r redis.Cmdable, start time.Time, days int) error {
	since, err := r.Get(ctx, StartedKey).Int64()
	if errors.Is(err, redis.Nil) {
		return ErrNotReady
	}
	if err != nil {
		return err
	}
	if start.Before(Day(time.Unix(since, 0)).AddDate(0, 0, 1)) {
		return ErrNotReady
	}
	keys := make([]string, days)
	for i := range keys {
		keys[i] = Key(start.AddDate(0, 0, i))
	}
	n, err := r.Exists(ctx, keys...).Result()
	if err != nil {
		return err
	}
	if n != int64(days) {
		return ErrNotReady
	}
	return nil
}
