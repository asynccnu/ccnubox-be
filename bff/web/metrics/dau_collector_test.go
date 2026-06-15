package metrics

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/redis/go-redis/v9"
)

func newTestRedis(t *testing.T) (*miniredis.Miniredis, redis.Cmdable) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis failed: %v", err)
	}
	t.Cleanup(mr.Close)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return mr, client
}

func newTestDesc() *prometheus.Desc {
	return prometheus.NewDesc(
		"ccnubox_dau",
		"test desc",
		[]string{"date"},
		nil,
	)
}

// nopLogger 静默 logger, 测试中不期望日志输出.
type nopLogger struct{}

func (nopLogger) WithContext(context.Context) logger.Logger { return nopLogger{} }
func (nopLogger) With(...logger.Field) logger.Logger        { return nopLogger{} }
func (nopLogger) Debug(string, ...logger.Field)             {}
func (nopLogger) Info(string, ...logger.Field)              {}
func (nopLogger) Warn(string, ...logger.Field)              {}
func (nopLogger) Error(string, ...logger.Field)             {}
func (nopLogger) Debugf(string, ...interface{})             {}
func (nopLogger) Infof(string, ...interface{})              {}
func (nopLogger) Warnf(string, ...interface{})              {}
func (nopLogger) Errorf(string, ...interface{})             {}
func (nopLogger) AddCallerSkip(int) logger.Logger           { return nopLogger{} }

var _ logger.Logger = nopLogger{}

func TestDAUCollector_Describe(t *testing.T) {
	_, client := newTestRedis(t)
	desc := newTestDesc()
	c := NewDAUCollector(desc, client, nopLogger{})

	ch := make(chan *prometheus.Desc, 1)
	c.Describe(ch)
	close(ch)

	got := <-ch
	if got == nil {
		t.Fatal("Describe did not emit any desc")
	}
	if got.String() != desc.String() {
		t.Fatalf("Describe emitted wrong desc:\n got:  %s\n want: %s", got, desc)
	}
}

func TestDAUCollector_Collect_HappyPath(t *testing.T) {
	mr, client := newTestRedis(t)
	desc := newTestDesc()
	c := NewDAUCollector(desc, client, nopLogger{})

	// 预填 "今天" 和 "昨天" 两天的 HLL, 每个桶各加 1 个 studentId。
	today := time.Now().Local().Format("2006-01-02")
	yesterday := time.Now().Local().AddDate(0, 0, -1).Format("2006-01-02")

	mr.PfAdd("dau:"+today+"-00-00", "stu1", "stu2")
	mr.PfAdd("dau:"+today+"-12-30", "stu3")
	mr.PfAdd("dau:"+yesterday+"-08-15", "stu4", "stu5", "stu6")

	ch := make(chan prometheus.Metric, 64)
	c.Collect(ch)
	close(ch)

	got := make(map[string]float64)
	for m := range ch {
		// 解析 metric 的 label 和 value
		var pb dto.Metric
		if err := m.Write(&pb); err != nil {
			t.Fatalf("write metric: %v", err)
		}
		if len(pb.Label) != 1 {
			t.Fatalf("expected 1 label, got %d", len(pb.Label))
		}
		if pb.Label[0].GetName() != "date" {
			t.Fatalf("expected label 'date', got %q", pb.Label[0].GetName())
		}
		got[pb.Label[0].GetValue()] = pb.GetGauge().GetValue()
	}

	if v := got[today]; v != 3 {
		t.Errorf("today DAU: got %v, want 3", v)
	}
	if v := got[yesterday]; v != 3 {
		t.Errorf("yesterday DAU: got %v, want 3", v)
	}
}

func TestDAUCollector_Collect_PartialDay(t *testing.T) {
	mr, client := newTestRedis(t)
	desc := newTestDesc()
	c := NewDAUCollector(desc, client, nopLogger{})

	// 只填今天 5 个桶 (96 桶里的 5 个), 每个桶 1 个不同 studentId
	today := time.Now().Local().Format("2006-01-02")
	for _, b := range []string{"00-00", "00-15", "00-30", "00-45", "01-00"} {
		mr.PfAdd("dau:"+today+"-"+b, "stu-"+b)
	}

	ch := make(chan prometheus.Metric, 64)
	c.Collect(ch)
	close(ch)

	var got float64
	var found bool
	for m := range ch {
		var p dto.Metric
		if err := m.Write(&p); err != nil {
			t.Fatalf("write metric: %v", err)
		}
		if len(p.Label) == 1 && p.Label[0].GetName() == "date" && p.Label[0].GetValue() == today {
			found = true
			got = p.GetGauge().GetValue()
		}
	}
	if !found {
		t.Fatalf("expected today metric to be emitted")
	}
	// 5 个桶 × 1 个 studentId = 5 唯一; HLL 对 ≤32 唯一是精确的, 期望 5
	if got != 5 {
		t.Errorf("today DAU: got %v, want 5", got)
	}
}

func TestDAUCollector_Collect_RedisDown(t *testing.T) {
	mr, client := newTestRedis(t)

	// 先正常填今天和昨天的数据, 走一次成功 Collect 把缓存写进 values
	today := time.Now().Local().Format("2006-01-02")
	yesterday := time.Now().Local().AddDate(0, 0, -1).Format("2006-01-02")
	mr.PfAdd("dau:"+today+"-00-00", "stu1")
	mr.PfAdd("dau:"+yesterday+"-00-00", "stu2", "stu3")

	desc := newTestDesc()
	c := NewDAUCollector(desc, client, nopLogger{})

	ch1 := make(chan prometheus.Metric, 64)
	c.Collect(ch1)
	close(ch1)
	for range ch1 { // drain
	}

	// 现在杀掉 redis, 再次 Collect, 应该 emit 上次成功的缓存值 (不 panic, 不掉 0)
	mr.Close()

	ch2 := make(chan prometheus.Metric, 64)
	c.Collect(ch2)
	close(ch2)

	got := make(map[string]float64)
	for m := range ch2 {
		var p dto.Metric
		if err := m.Write(&p); err != nil {
			t.Fatalf("write metric: %v", err)
		}
		if len(p.Label) == 1 && p.Label[0].GetName() == "date" {
			got[p.Label[0].GetValue()] = p.GetGauge().GetValue()
		}
	}
	if v := got[today]; v != 1 {
		t.Errorf("today DAU after redis down: got %v, want 1 (cached)", v)
	}
	if v := got[yesterday]; v != 2 {
		t.Errorf("yesterday DAU after redis down: got %v, want 2 (cached)", v)
	}
}

func TestDAUCollector_Collect_RollingWindow(t *testing.T) {
	mr, client := newTestRedis(t)
	desc := newTestDesc()
	c := NewDAUCollector(desc, client, nopLogger{})

	// 预填 35 天的数据, 验证只 emit 最近 30 天
	today := time.Now().Local()
	for offset := 0; offset < 35; offset++ {
		date := today.AddDate(0, 0, -offset).Format("2006-01-02")
		// 给每天的 00-00 桶加 1 个 studentId, 标记 offset 用作调试
		mr.PfAdd(fmt.Sprintf("dau:%s-00-00", date), fmt.Sprintf("stu-%d", offset))
	}

	ch := make(chan prometheus.Metric, 128)
	c.Collect(ch)
	close(ch)

	count := 0
	for range ch {
		count++
	}
	if count != dauWindowDays {
		t.Errorf("expected %d emitted metrics, got %d", dauWindowDays, count)
	}

	// 验证 values map 在 30 天窗口内
	c.mu.RLock()
	valuesSize := len(c.values)
	c.mu.RUnlock()
	if valuesSize > dauWindowDays {
		t.Errorf("values map too large: got %d, want <= %d", valuesSize, dauWindowDays)
	}
}

func TestDAUCollector_Collect_TempKeyTTL(t *testing.T) {
	mr, client := newTestRedis(t)
	desc := newTestDesc()
	c := NewDAUCollector(desc, client, nopLogger{})

	today := time.Now().Local().Format("2006-01-02")
	mr.PfAdd("dau:"+today+"-00-00", "stu1")

	ch := make(chan prometheus.Metric, 64)
	c.Collect(ch)
	close(ch)
	for range ch { // drain
	}

	// 临时 key 应存在, 且 TTL 在 (0, 60s] 之间
	tempKey := "dau:tmp:" + today
	if !mr.Exists(tempKey) {
		t.Fatalf("expected temp key %q to exist after Collect", tempKey)
	}
	ttl := mr.TTL(tempKey)
	if ttl <= 0 || ttl > 60*time.Second {
		t.Errorf("expected temp key TTL in (0, 60s], got %v", ttl)
	}
}
