package cron

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/go-redsync/redsync/v4"
	redsyncredis "github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

func newTestRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis failed: %v", err)
	}
	t.Cleanup(mr.Close)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return mr, client
}

func newTestRedsync(client *redis.Client) *redsync.Redsync {
	return redsync.New(redsyncredis.NewPool(client))
}

func newTestGauge() prometheus.Gauge {
	return prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "test_dau",
		Help: "test",
	})
}

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

func readGauge(t *testing.T, g prometheus.Gauge) float64 {
	t.Helper()
	reg := prometheus.NewRegistry()
	if err := reg.Register(g); err != nil {
		var alreadyRegistered prometheus.AlreadyRegisteredError
		if !errors.As(err, &alreadyRegistered) {
			t.Fatalf("register gauge: %v", err)
		}
	}
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	for _, mf := range mfs {
		for _, m := range mf.GetMetric() {
			return m.GetGauge().GetValue()
		}
	}
	return 0
}

// TestDAURefresher_Refresh_Ok 验证 Refresh 会 PFMERGE 昨天 96 桶 → PFCOUNT
// gauge 写入正确值, 同时 Redis dau:latest 兜底也被更新
func TestDAURefresher_Refresh_Ok(t *testing.T) {
	mr, client := newTestRedis(t)
	rs := newTestRedsync(client)
	gauge := newTestGauge()
	d := NewDAURefresher(client, gauge, rs, nopLogger{})

	yesterday := time.Now().Local().AddDate(0, 0, -1).Format("2006-01-02")
	// 预填 3 个不同学号到 3 个不同桶, 期望 HLL 计数 = 3。
	for i, b := range []string{"00-00", "12-30", "23-45"} {
		mr.PfAdd(fmt.Sprintf("dau:%s-%s", yesterday, b), fmt.Sprintf("stu-%d", i))
	}

	d.Refresh(context.Background())

	if v := readGauge(t, gauge); v != 3 {
		t.Errorf("gauge: got %v, want 3", v)
	}
	v, err := client.Get(context.Background(), dauLatestKey).Int64()
	if err != nil {
		t.Fatalf("read dau:latest: %v", err)
	}
	if v != 3 {
		t.Errorf("dau:latest: got %d, want 3", v)
	}
}

// TestDAURefresher_Refresh_EmptyDay 没有 HLL 数据时, Refresh 不应 panic
// gauge 写 0, Redis 兜底写 0
func TestDAURefresher_Refresh_EmptyDay(t *testing.T) {
	_, client := newTestRedis(t)
	rs := newTestRedsync(client)
	gauge := newTestGauge()
	d := NewDAURefresher(client, gauge, rs, nopLogger{})

	d.Refresh(context.Background())

	if v := readGauge(t, gauge); v != 0 {
		t.Errorf("gauge: got %v, want 0", v)
	}
}

// TestDAURefresher_Bootstrap_Ok 验证 Bootstrap 从 Redis 恢复值到 gauge
func TestDAURefresher_Bootstrap_Ok(t *testing.T) {
	mr, client := newTestRedis(t)
	rs := newTestRedsync(client)
	gauge := newTestGauge()
	d := NewDAURefresher(client, gauge, rs, nopLogger{})

	mr.Set(dauLatestKey, "456")

	d.Bootstrap(context.Background())

	if v := readGauge(t, gauge); v != 456 {
		t.Errorf("gauge: got %v, want 456", v)
	}
}

// TestDAURefresher_Bootstrap_NoKey 验证 Redis 没有 dau:latest 时 Bootstrap 不 panic, gauge 保持 0
func TestDAURefresher_Bootstrap_NoKey(t *testing.T) {
	_, client := newTestRedis(t)
	rs := newTestRedsync(client)
	gauge := newTestGauge()
	d := NewDAURefresher(client, gauge, rs, nopLogger{})

	d.Bootstrap(context.Background())

	if v := readGauge(t, gauge); v != 0 {
		t.Errorf("gauge: got %v, want 0", v)
	}
}
