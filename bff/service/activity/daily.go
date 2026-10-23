package activity

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	RetentionDays = 32
	// Old best-effort sets cannot be certified retroactively.
	StartedKey = "activity:daily:v2:started_at"
)

// China is independent of the Pod's local timezone and needs no host tzdata.
var China = time.FixedZone("Asia/Shanghai", 8*60*60)

var ErrNotReady = errors.New("活跃历史不足或日期数据缺失，请积累完整自然日数据后重试")
var ErrDayClosed = errors.New("activity day is already complete")

func Day(t time.Time) time.Time {
	t = t.In(China)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, China)
}

func Key(t time.Time) string         { return "activity:daily:v2:" + t.In(China).Format("2006-01-02") }
func CompleteKey(t time.Time) string { return Key(t) + ":complete" }
func pendingKey(t time.Time) string  { return Key(t) + ":pending" }
func countKey(t time.Time) string    { return Key(t) + ":accepted" }

// Register an intent before admitting an authenticated request. The pending set
// survives a worker failure/restart. accepted counts unique admitted IDs, rather
// than inferring completeness from the number of members that happened to land.
const recordScript = `
if redis.call('EXISTS', KEYS[5]) == 1 then return -1 end
local count = redis.call('GET', KEYS[4])
local members = redis.call('SCARD', KEYS[2])
local pending = redis.call('SCARD', KEYS[3])
if count then
    if members == 0 or members + pending ~= tonumber(count) + 1 then
        return redis.error_reply('activity membership or pending intents lost')
    end
else
    if members ~= 0 or pending ~= 0 then
        return redis.error_reply('activity admission count lost')
    end
    redis.call('SET', KEYS[4], 0)
    redis.call('SADD', KEYS[2], '')
end
redis.call('SET', KEYS[1], ARGV[2], 'NX')
if ARGV[1] ~= '' and redis.call('SISMEMBER', KEYS[2], ARGV[1]) == 0 then
    if redis.call('SADD', KEYS[3], ARGV[1]) == 1 then
        redis.call('INCR', KEYS[4])
    end
end
for i = 2, 4 do redis.call('EXPIREAT', KEYS[i], ARGV[3]) end
return 1
`

// Record durably queues exact membership. Callers must not admit an
// authenticated request unless this succeeds; approximate metrics are separate.
func Record(ctx context.Context, r redis.Cmdable, studentID string, now time.Time) error {
	n, err := r.Eval(ctx, recordScript, []string{StartedKey, Key(now), pendingKey(now), countKey(now), CompleteKey(now)},
		studentID, now.Unix(), Day(now).AddDate(0, 0, RetentionDays).Unix()).Int()
	if err == nil && n == -1 {
		return ErrDayClosed
	}
	return err
}

// Move a bounded batch without removing an intent until its member is stored.
// Sealing shares the same Redis serialization boundary as admission, so a late
// writer is either included before the seal or rejected by Record afterwards.
const flushScript = `
local count = redis.call('GET', KEYS[3])
if not count then return 0 end
local members = redis.call('SCARD', KEYS[1])
local pending = redis.call('SCARD', KEYS[2])
if members == 0 or members + pending ~= tonumber(count) + 1 then
    return redis.error_reply('activity membership or pending intents lost')
end
local ids = redis.call('SRANDMEMBER', KEYS[2], 512)
if #ids > 0 then
    redis.call('SADD', KEYS[1], unpack(ids))
    redis.call('SREM', KEYS[2], unpack(ids))
end
redis.call('EXPIREAT', KEYS[1], ARGV[2])
pending = redis.call('SCARD', KEYS[2])
if pending == 0 and ARGV[1] == '1' then
    redis.call('SET', KEYS[4], count, 'EXAT', ARGV[2])
end
return pending
`

// EnsureDay replays durable intents and certifies only closed natural days after
// every admitted member has landed. Existing date keys alone certify nothing.
// Missing/corrupt history is left unavailable, never reconstructed as empty.
func EnsureDay(ctx context.Context, r redis.Cmdable, now time.Time) error {
	if err := Record(ctx, r, "", now); err != nil {
		return err
	}
	var failures []error
	for i := 0; i < RetentionDays; i++ {
		day := Day(now).AddDate(0, 0, -i)
		closed := 0
		if i > 0 {
			closed = 1
		}
		for {
			pending, err := r.Eval(ctx, flushScript, []string{Key(day), pendingKey(day), countKey(day), CompleteKey(day)},
				closed, day.AddDate(0, 0, RetentionDays).Unix()).Int()
			if err != nil {
				failures = append(failures, fmt.Errorf("flush activity %s: %w", day.Format("2006-01-02"), err))
				break
			}
			if pending == 0 {
				break
			}
		}
		if ctx.Err() != nil {
			break
		}
	}
	return errors.Join(failures...)
}

// ReadDay validates the completion count and snapshots the members in one Redis
// operation. A set partially deleted after CheckWindow must not freeze a partial
// recipient list. The empty sentinel distinguishes a complete zero-user day.
func ReadDay(ctx context.Context, r redis.Cmdable, day time.Time) ([]string, error) {
	return r.Eval(ctx, `
local count = redis.call('GET', KEYS[2])
if not count then return {} end
local expected = tonumber(count)
if not expected or expected < 0 then
    return redis.error_reply('invalid activity completion count')
end
local size = redis.call('SCARD', KEYS[1])
if size ~= expected + 1 then return {} end
if size > 100001 then return redis.error_reply('单日活跃候选人数超过 100000') end
return redis.call('SMEMBERS', KEYS[1])
`, []string{Key(day), CompleteKey(day)}).StringSlice()
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
		return fmt.Sprintf("%s；以下日期的用户活跃明细缺失或尚未完成：%s。请检查待处理明细、采集是否中断或 Redis 数据是否被清理", window, strings.Join(e.MissingDays, "、"))
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
	checks := make([]*redis.StringCmd, days)
	counts := make([]*redis.IntCmd, days)
	for i := range checks {
		day := start.AddDate(0, 0, i)
		checks[i] = pipe.Get(ctx, CompleteKey(day))
		counts[i] = pipe.SCard(ctx, Key(day))
	}
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return err
	}
	e := historyError(start, days, "missing_dates")
	for i, check := range checks {
		value, err := check.Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			return err
		}
		expected, parseErr := strconv.ParseInt(value, 10, 64)
		if err == nil && (parseErr != nil || expected < 0) {
			return fmt.Errorf("invalid activity completion marker: %q", value)
		}
		if counts[i].Err() != nil {
			return counts[i].Err()
		}
		if errors.Is(err, redis.Nil) || counts[i].Val() != expected+1 {
			e.MissingDays = append(e.MissingDays, Day(start).AddDate(0, 0, i).Format("2006-01-02"))
		}
	}
	if len(e.MissingDays) > 0 {
		return e
	}
	return nil
}
