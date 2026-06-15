package metrics

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"

	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

const (
	dauWindowDays    = 30
	dauBucketMinutes = 15
	dauBucketsPerDay = 24 * 60 / dauBucketMinutes // 96
	dauScrapeTimeout = 3 * time.Second
)

// DAUCollector 每次抓取时从 Redis 拉取最近 30 天的日活并 emit 给 promhttp。
// 多 Pod 部署时所有 Pod 共享 Redis, 计算结果一致; Prometheus 端需用 avg() 或 min() 去重。
type DAUCollector struct {
	desc        *prometheus.Desc
	redisClient redis.Cmdable
	logger      logger.Logger

	mu     sync.RWMutex
	values map[string]float64 // date -> last known value, 用于 Redis 失败时仍能 emit 上次值
}

func NewDAUCollector(desc *prometheus.Desc, redisClient redis.Cmdable, l logger.Logger) *DAUCollector {
	return &DAUCollector{
		desc:        desc,
		redisClient: redisClient,
		logger:      l,
		values:      make(map[string]float64),
	}
}

func (c *DAUCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

func (c *DAUCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), dauScrapeTimeout)
	defer cancel()

	today := time.Now().Local()
	keepDates := make(map[string]struct{}, dauWindowDays)
	successDays := 0

	for offset := 0; offset < dauWindowDays; offset++ {
		date := today.AddDate(0, 0, -offset).Format("2006-01-02")
		keepDates[date] = struct{}{}

		count, err := c.countDay(ctx, date)
		if err != nil {
			c.logger.Warn("dau collector: failed to count day",
				logger.String("date", date),
				logger.String("err", err.Error()))
			c.emitLastKnown(ch, date)
			continue
		}

		c.mu.Lock()
		c.values[date] = float64(count)
		c.mu.Unlock()
		ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, float64(count), date)
		successDays++
	}

	// 清扫滑出 30 天窗口的旧 date, 防止 values 无限增长。
	c.mu.Lock()
	for d := range c.values {
		if _, keep := keepDates[d]; !keep {
			delete(c.values, d)
		}
	}
	c.mu.Unlock()

	if successDays < dauWindowDays {
		c.logger.Warn("dau scrape partial failure",
			logger.Int("success", successDays),
			logger.Int("total", dauWindowDays))
	}
}

// emitLastKnown 在 Redis 失败时, 把缓存的上一次成功值 emit 出去, 避免指标突然掉 0。
// 若从未成功过 (values 里没有该 date), 则跳过, 让 Prometheus 显示为空。
func (c *DAUCollector) emitLastKnown(ch chan<- prometheus.Metric, date string) {
	c.mu.RLock()
	v, ok := c.values[date]
	c.mu.RUnlock()
	if !ok {
		return
	}
	ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, v, date)
}

// countDay 把指定日期的 96 个 15min 桶 PFMERGE 到临时 key, PFCOUNT 后用 TTL 兜底清理。
// 失败时返回 error, Collect 跳过那一天, 其它天不受影响。
func (c *DAUCollector) countDay(ctx context.Context, date string) (int64, error) {
	tempKey := "dau:tmp:" + date

	srcKeys := make([]string, 0, dauBucketsPerDay)
	for h := 0; h < 24; h++ {
		for m := 0; m < 60; m += dauBucketMinutes {
			srcKeys = append(srcKeys, fmt.Sprintf("dau:%s-%02d-%02d", date, h, m))
		}
	}

	pipe := c.redisClient.Pipeline()
	pipe.PFMerge(ctx, tempKey, srcKeys...)
	pfcountCmd := pipe.PFCount(ctx, tempKey)
	pipe.Expire(ctx, tempKey, 60*time.Second) // 兜底 TTL, 60s 后自动清理
	// 注: 不主动 DEL, 避免 DEL 失败时多一条错误日志。60s TTL 足够小, 不影响下次抓取。

	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	return pfcountCmd.Val(), nil
}
