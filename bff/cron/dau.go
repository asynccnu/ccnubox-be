package cron

import (
	"context"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/go-redsync/redsync/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

const (
	dauLockKey   = "dau:daily:lock"
	dauLatestKey = "dau:latest"
	dauBucketMin = 15
)

// DAURefresher 负责每天定时把昨天的 DAU 写入 Prometheus Gauge
// 用 redsync 拿分布式锁, 避免多 BFF Pod 重复计算与重复 Set
// 写完 gauge 顺便 SET dau:latest 到 Redis, 启动时 Bootstrap 用作重启兜底
// PFMERGE 96 个 15min 桶到临时 key 再 PFCOUNT, 临时 key 用 60s TTL 自清
type DAURefresher struct {
	redis redis.Cmdable
	gauge prometheus.Gauge
	rs    *redsync.Redsync
	log   logger.Logger
}

func NewDAURefresher(r redis.Cmdable, g prometheus.Gauge, rs *redsync.Redsync, l logger.Logger) *DAURefresher {
	return &DAURefresher{redis: r, gauge: g, rs: rs, log: l}
}

func (d *DAURefresher) Refresh(ctx context.Context) {
	mu := d.rs.NewMutex(dauLockKey, redsync.WithExpiry(2*time.Minute))
	if err := mu.LockContext(ctx); err != nil {
		d.log.Warn("dau refresh: acquire lock failed", logger.Error(err))
		return
	}
	defer func() {
		if _, err := mu.UnlockContext(ctx); err != nil {
			d.log.Warn("dau refresh: release lock failed", logger.Error(err))
		}
	}()

	yesterday := time.Now().Local().AddDate(0, 0, -1).Format("2006-01-02")
	count, err := d.countDay(ctx, yesterday)
	if err != nil {
		d.log.Error("dau refresh: count failed",
			logger.String("date", yesterday), logger.Error(err))
		return
	}

	d.gauge.Set(float64(count))
	d.log.Info("dau refresh ok",
		logger.String("date", yesterday), logger.Int64("count", count))

	// 兜底写 Redis 启动时 Bootstrap 从这里恢复, 防止 BFF 滚动重启出现 0 点
	if err := d.redis.Set(ctx, dauLatestKey, count, 7*24*time.Hour).Err(); err != nil {
		d.log.Warn("dau refresh: write latest failed", logger.Error(err))
	}
}

// Bootstrap 启动时从 Redis 恢复最近一次成功的 DAU 值到 gauge
// 若 Redis 没有, 保持 0, 等首次 Refresh 触发
func (d *DAURefresher) Bootstrap(ctx context.Context) {
	v, err := d.redis.Get(ctx, dauLatestKey).Int64()
	if err != nil {
		d.log.Info("dau bootstrap: no previous value", logger.Error(err))
		return
	}
	d.gauge.Set(float64(v))
	d.log.Info("dau bootstrap ok", logger.Int64("count", v))
}

// countDay 把指定日期的 96 个 15min 桶 PFMERGE 到临时 key, PFCOUNT 后用 TTL 兜底清理
func (d *DAURefresher) countDay(ctx context.Context, date string) (int64, error) {
	tempKey := "dau:tmp:" + date

	srcKeys := make([]string, 0, 24*60/dauBucketMin)
	for h := 0; h < 24; h++ {
		for m := 0; m < 60; m += dauBucketMin {
			srcKeys = append(srcKeys, fmt.Sprintf("dau:%s-%02d-%02d", date, h, m))
		}
	}

	pipe := d.redis.Pipeline()
	pipe.PFMerge(ctx, tempKey, srcKeys...)
	pfcountCmd := pipe.PFCount(ctx, tempKey)
	pipe.Expire(ctx, tempKey, 60*time.Second)

	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	return pfcountCmd.Val(), nil
}
