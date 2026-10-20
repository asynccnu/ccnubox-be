package activity

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

// HistoryNotReadyError explains why exact daily user records cannot yet be
// used. It wraps ErrNotReady to preserve the existing HTTP 409 / 40804 contract.
type HistoryNotReadyError struct {
	Reason              string
	WindowStart         string
	WindowEnd           string
	CollectionStartedAt int64
	FirstCompleteDay    string
	EarliestReadyDate   string
	MissingDays         []string
}

func (e *HistoryNotReadyError) Unwrap() error { return ErrNotReady }
func (e *HistoryNotReadyError) Error() string {
	window := fmt.Sprintf("需要北京时间 %s 至 %s 的每日用户活跃明细", e.WindowStart, e.WindowEnd)
	switch e.Reason {
	case "not_started":
		return "尚未记录每日活跃学号；" + window + "。Prometheus 日活人数不能替代学号明细，请确认 BFF 明细采集已部署运行"
	case "collecting":
		since := time.Unix(e.CollectionStartedAt, 0).In(China).Format("2006-01-02 15:04:05")
		return fmt.Sprintf("%s；学号明细从 %s 开始采集，首个完整自然日为 %s，按当前窗口最早于 %s 可用（北京时间）", window, since, e.FirstCompleteDay, e.EarliestReadyDate)
	default:
		return fmt.Sprintf("%s；缺少以下日期的用户活跃明细：%s。请检查采集是否中断或 Redis 数据是否被清理", window, strings.Join(e.MissingDays, "、"))
	}
}

func historyError(start time.Time, days int, reason string) *HistoryNotReadyError {
	return &HistoryNotReadyError{Reason: reason, WindowStart: Day(start).Format("2006-01-02"), WindowEnd: Day(start).AddDate(0, 0, days-1).Format("2006-01-02")}
}

// MissingDay reports a day lost between the window check and reading its set.
func MissingDay(start time.Time, days int, day time.Time) error {
	e := historyError(start, days, "missing_dates")
	e.MissingDays = []string{Day(day).Format("2006-01-02")}
	return e
}

func CheckWindow(ctx context.Context, r redis.Cmdable, start time.Time, days int) error {
	since, err := r.Get(ctx, StartedKey).Int64()
	if errors.Is(err, redis.Nil) {
		return historyError(start, days, "not_started")
	}
	if err != nil {
		return err
	}
	first := Day(time.Unix(since, 0)).AddDate(0, 0, 1)
	if Day(start).Before(first) {
		e := historyError(start, days, "collecting")
		e.CollectionStartedAt = since
		e.FirstCompleteDay = first.Format("2006-01-02")
		e.EarliestReadyDate = first.AddDate(0, 0, days).Format("2006-01-02")
		return e
	}
	pipe := r.Pipeline()
	checks := make([]*redis.IntCmd, days)
	for i := range checks {
		checks[i] = pipe.Exists(ctx, Key(start.AddDate(0, 0, i)))
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}
	e := historyError(start, days, "missing_dates")
	for i, check := range checks {
		if check.Val() == 0 {
			e.MissingDays = append(e.MissingDays, Day(start).AddDate(0, 0, i).Format("2006-01-02"))
		}
	}
	if len(e.MissingDays) > 0 {
		return e
	}
	return nil
}
