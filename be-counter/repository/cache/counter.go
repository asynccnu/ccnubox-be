package cache

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/redis/go-redis/v9"
)

const RedisPrefix = "ccnubox:FUC"

const (
	ExpireTime = 7 * 24 * time.Hour // 每日Hash过期时间
	DedupTime  = 1 * time.Hour      // 1h去重窗口
)

// 每日hash存储当日活跃度完整数据
func dailyHashKey(t time.Time) string {
	return fmt.Sprintf("%s:%s", RedisPrefix, t.Format("2006-01-02"))
}

// string标记去重
func dedupKey(studentId string) string {
	return fmt.Sprintf("%s:dedup:%s", RedisPrefix, studentId)
}

// ZSet聚合综合活跃度
func aggZSetKey() string {
	return RedisPrefix + ":aggregated"
}

type CounterCache interface {
	AddCounter(ctx context.Context, studentId string) (bool, error)
	RebuildCounter(ctx context.Context) error
	DecayCounter(ctx context.Context, studentIds []string) error
	BoostScores(ctx context.Context, studentIds []string) error
	GetStudentCounter(ctx context.Context, studentId string) (int64, error)
	GetCounterByRank(ctx context.Context, start, stop int64) ([]string, error)
	GetCounterCount(ctx context.Context) (int64, error)
}

type RedisCounterCache struct {
	cmd redis.Cmdable
}

func NewRedisCounterCache(cmd redis.Cmdable) CounterCache {
	return &RedisCounterCache{cmd: cmd}
}

// AddCounter 活跃度+1
func (c *RedisCounterCache) AddCounter(ctx context.Context, studentId string) (bool, error) {
	now := time.Now()
	key := dailyHashKey(now)
	if _, err := c.cmd.HIncrBy(ctx, key, studentId, 1).Result(); err != nil {
		return false, errorx.Errorf("cache: hincrby failed, studentId: %s, err: %w", studentId, err)
	}
	_, _ = c.cmd.Expire(ctx, key, ExpireTime).Result()
	return true, nil
}

// RebuildCounter 聚合近7天Hash，重建ZSet
func (c *RedisCounterCache) RebuildCounter(ctx context.Context) error {
	now := time.Now()
	pipe := c.cmd.Pipeline()
	cmds := make([]*redis.MapStringStringCmd, 7)
	for i := 0; i < 7; i++ {
		cmds[i] = pipe.HGetAll(ctx, dailyHashKey(now.AddDate(0, 0, -i)))
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return errorx.Errorf("cache: rebuild hgetall failed: %w", err)
	}

	totals := make(map[string]int64)
	for _, cmd := range cmds {
		result, err := cmd.Result()
		if err != nil || result == nil {
			continue
		}
		for sid, s := range result {
			v, _ := strconv.ParseInt(s, 10, 64)
			totals[sid] += v
		}
	}
	if len(totals) == 0 {
		return nil
	}

	pipe2 := c.cmd.TxPipeline()
	pipe2.Del(ctx, aggZSetKey())
	for sid, sum := range totals {
		if sum <= 0 {
			continue
		}
		pipe2.ZAdd(ctx, aggZSetKey(), redis.Z{
			Member: sid,
			Score:  float64(sum),
		})
	}
	_, err := pipe2.Exec(ctx)
	if err != nil {
		return errorx.Errorf("cache: rebuild zadd failed: %w", err)
	}
	return nil
}

// DecayCounter 活跃度衰减
func (c *RedisCounterCache) DecayCounter(ctx context.Context, studentIds []string) error {
	if len(studentIds) == 0 {
		return nil
	}

	now := time.Now()
	todayKey := dailyHashKey(now)

	scores, err := c.cmd.ZMScore(ctx, aggZSetKey(), studentIds...).Result()
	if err != nil {
		return errorx.Errorf("cache: zmscore failed: %w", err)
	}

	pipe := c.cmd.Pipeline()
	hasWork := false

	for i, sid := range studentIds {
		score := int64(0)
		if i < len(scores) {
			score = int64(scores[i])
		}
		if score <= 0 {
			continue
		}
		//拟合的一个活跃度衰减的指数函数，活跃度越高降低的幅度越大，衰减率控制在[5%,30%]
		rate := 0.05 + 0.25*(1-math.Exp(-float64(score)/20))
		decay := calcVariation(score, rate)
		if decay <= 0 {
			continue
		}
		pipe.HIncrBy(ctx, todayKey, sid, -decay)
		hasWork = true
	}
	if !hasWork {
		return nil
	}
	pipe.Expire(ctx, todayKey, ExpireTime)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return errorx.Errorf("cache: decay failed: %w", err)
	}
	return nil
}

// calcVariation 变化量 = ceil(score * rate)
func calcVariation(score int64, rate float64) int64 {
	if score <= 0 || rate <= 0 {
		return 0
	}
	decay := int64(math.Ceil(float64(score) * rate))

	return min(decay, score)
}

// BoostScores 为指定学生提升综合活跃度的20%
func (c *RedisCounterCache) BoostScores(ctx context.Context, studentIds []string) error {
	if len(studentIds) == 0 {
		return nil
	}

	// 批量去重
	pipe := c.cmd.Pipeline()
	cmds := make([]*redis.BoolCmd, len(studentIds))
	for i, sid := range studentIds {
		cmds[i] = pipe.SetNX(ctx, dedupKey(sid), "1", DedupTime)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return errorx.Errorf("cache: boost dedup failed: %w", err)
	}

	var needBoost []string
	for i, cmd := range cmds {
		ok, _ := cmd.Result()
		if ok {
			needBoost = append(needBoost, studentIds[i])
		}
	}
	if len(needBoost) == 0 {
		return nil
	}

	// 读当前分数
	scores, err := c.cmd.ZMScore(ctx, aggZSetKey(), needBoost...).Result()
	if err != nil {
		return errorx.Errorf("cache: boost zmscore failed: %w", err)
	}

	// 提升综合活跃度
	now := time.Now()
	todayKey := dailyHashKey(now)
	pipe2 := c.cmd.Pipeline()
	hasWork := false
	for i, sid := range needBoost {
		score := int64(0)
		if i < len(scores) {
			score = int64(scores[i])
		}
		//和降低的函数相反，分数越高提升的越少，分数越低提升的越多
		rate := 0.3 - 0.25*(1-math.Exp(-float64(score)/20))
		boost := calcVariation(score, rate)
		if boost <= 0 {
			continue
		}
		pipe2.HIncrBy(ctx, todayKey, sid, boost)
		hasWork = true
	}
	if !hasWork {
		return nil
	}
	pipe2.Expire(ctx, todayKey, ExpireTime)
	_, err = pipe2.Exec(ctx)
	if err != nil {
		return errorx.Errorf("cache: boost write failed: %w", err)
	}
	return nil
}

// GetCounter 获取单个学生聚合活跃度
func (c *RedisCounterCache) GetStudentCounter(ctx context.Context, studentId string) (int64, error) {
	s, err := c.cmd.ZScore(ctx, aggZSetKey(), studentId).Result()
	if err != nil {
		if errorx.Is(err, redis.Nil) {
			return 0, nil
		}
		return 0, errorx.Errorf("cache: zscore failed, studentId: %s, err: %w", studentId, err)
	}
	return int64(s), nil
}

// GetAllCounter 获取所有学生聚合活跃度
func (c *RedisCounterCache) GetAllCounter(ctx context.Context) ([]*Counter, error) {
	results, err := c.cmd.ZRangeWithScores(ctx, aggZSetKey(), 0, -1).Result()
	if err != nil {
		return nil, errorx.Errorf("cache: zrange with scores failed: %w", err)
	}
	out := make([]*Counter, 0, len(results))
	for _, z := range results {
		out = append(out, &Counter{
			StudentId: z.Member.(string),
			Count:     int64(z.Score),
		})
	}
	return out, nil
}

// GetCounterByRank 按排名区间取学生ID
func (c *RedisCounterCache) GetCounterByRank(ctx context.Context, start, stop int64) ([]string, error) {
	members, err := c.cmd.ZRange(ctx, aggZSetKey(), start, stop).Result()
	if err != nil {
		return nil, errorx.Errorf("cache: zrange failed: %w", err)
	}
	return members, nil
}

func (c *RedisCounterCache) GetCounterCount(ctx context.Context) (int64, error) {
	count, err := c.cmd.ZCount(ctx, aggZSetKey(), "-inf", "+inf").Result()
	if err != nil {
		return 0, errorx.Errorf("cache: zcount failed: %w", err)
	}
	return count, nil
}

type Counter struct {
	StudentId string `json:"StudentId"`
	Count     int64  `json:"count"`
}
