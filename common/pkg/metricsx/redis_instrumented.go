package metricsx

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type InstrumentedRedis struct {
	redis.Cmdable
	metrics *Metrics
}

func NewInstrumentedRedis(cmd redis.Cmdable, metrics *Metrics) *InstrumentedRedis {
	return &InstrumentedRedis{
		Cmdable: cmd,
		metrics: metrics,
	}
}

func (r *InstrumentedRedis) observeOperation(operation string, duration time.Duration, err error) {
	status := "OK"
	if err != nil && !errors.Is(err, redis.Nil) {
		status = "Error"
		r.metrics.Redis.ErrorsTotal.WithLabelValues(operation, classifyRedisError(err)).Inc()
	}
	r.metrics.Redis.RequestsTotal.WithLabelValues(operation, status).Inc()
	r.metrics.Redis.Duration.WithLabelValues(operation).Observe(duration.Seconds())
}

func classifyRedisError(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	if errors.Is(err, context.Canceled) {
		return "canceled"
	}
	return "redis_error"
}

func (r *InstrumentedRedis) PFAdd(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	start := time.Now()
	cmd := r.Cmdable.PFAdd(ctx, key, values...)
	r.observeOperation("PFADD", time.Since(start), cmd.Err())
	return cmd
}

func (r *InstrumentedRedis) Get(ctx context.Context, key string) *redis.StringCmd {
	start := time.Now()
	cmd := r.Cmdable.Get(ctx, key)
	r.observeOperation("GET", time.Since(start), cmd.Err())
	return cmd
}

func (r *InstrumentedRedis) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	start := time.Now()
	cmd := r.Cmdable.Set(ctx, key, value, expiration)
	r.observeOperation("SET", time.Since(start), cmd.Err())
	return cmd
}

func (r *InstrumentedRedis) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd {
	start := time.Now()
	cmd := r.Cmdable.SetNX(ctx, key, value, expiration)
	r.observeOperation("SETNX", time.Since(start), cmd.Err())
	return cmd
}

func (r *InstrumentedRedis) HGet(ctx context.Context, key, field string) *redis.StringCmd {
	start := time.Now()
	cmd := r.Cmdable.HGet(ctx, key, field)
	r.observeOperation("HGET", time.Since(start), cmd.Err())
	return cmd
}

func (r *InstrumentedRedis) HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	start := time.Now()
	cmd := r.Cmdable.HSet(ctx, key, values...)
	r.observeOperation("HSET", time.Since(start), cmd.Err())
	return cmd
}

func (r *InstrumentedRedis) HGetAll(ctx context.Context, key string) *redis.MapStringStringCmd {
	start := time.Now()
	cmd := r.Cmdable.HGetAll(ctx, key)
	r.observeOperation("HGETALL", time.Since(start), cmd.Err())
	return cmd
}

func (r *InstrumentedRedis) MGet(ctx context.Context, keys ...string) *redis.SliceCmd {
	start := time.Now()
	cmd := r.Cmdable.MGet(ctx, keys...)
	r.observeOperation("MGET", time.Since(start), cmd.Err())
	return cmd
}

func (r *InstrumentedRedis) MSet(ctx context.Context, pairs ...interface{}) *redis.StatusCmd {
	start := time.Now()
	cmd := r.Cmdable.MSet(ctx, pairs...)
	r.observeOperation("MSET", time.Since(start), cmd.Err())
	return cmd
}

func (r *InstrumentedRedis) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	start := time.Now()
	cmd := r.Cmdable.Del(ctx, keys...)
	r.observeOperation("DEL", time.Since(start), cmd.Err())
	return cmd
}

func (r *InstrumentedRedis) Exists(ctx context.Context, keys ...string) *redis.IntCmd {
	start := time.Now()
	cmd := r.Cmdable.Exists(ctx, keys...)
	r.observeOperation("EXISTS", time.Since(start), cmd.Err())
	return cmd
}

func (r *InstrumentedRedis) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	start := time.Now()
	cmd := r.Cmdable.Expire(ctx, key, expiration)
	r.observeOperation("EXPIRE", time.Since(start), cmd.Err())
	return cmd
}

func (r *InstrumentedRedis) Incr(ctx context.Context, key string) *redis.IntCmd {
	start := time.Now()
	cmd := r.Cmdable.Incr(ctx, key)
	r.observeOperation("INCR", time.Since(start), cmd.Err())
	return cmd
}

func (r *InstrumentedRedis) Decr(ctx context.Context, key string) *redis.IntCmd {
	start := time.Now()
	cmd := r.Cmdable.Decr(ctx, key)
	r.observeOperation("DECR", time.Since(start), cmd.Err())
	return cmd
}

func (r *InstrumentedRedis) IncrBy(ctx context.Context, key string, value int64) *redis.IntCmd {
	start := time.Now()
	cmd := r.Cmdable.IncrBy(ctx, key, value)
	r.observeOperation("INCRBY", time.Since(start), cmd.Err())
	return cmd
}

func (r *InstrumentedRedis) DecrBy(ctx context.Context, key string, value int64) *redis.IntCmd {
	start := time.Now()
	cmd := r.Cmdable.DecrBy(ctx, key, value)
	r.observeOperation("DECRBY", time.Since(start), cmd.Err())
	return cmd
}

func (r *InstrumentedRedis) Append(ctx context.Context, key, value string) *redis.IntCmd {
	start := time.Now()
	cmd := r.Cmdable.Append(ctx, key, value)
	r.observeOperation("APPEND", time.Since(start), cmd.Err())
	return cmd
}

func (r *InstrumentedRedis) GetSet(ctx context.Context, key string, value interface{}) *redis.StringCmd {
	start := time.Now()
	cmd := r.Cmdable.GetSet(ctx, key, value)
	r.observeOperation("GETSET", time.Since(start), cmd.Err())
	return cmd
}

func (r *InstrumentedRedis) SMembers(ctx context.Context, key string) *redis.StringSliceCmd {
	start := time.Now()
	cmd := r.Cmdable.SMembers(ctx, key)
	r.observeOperation("SMEMBERS", time.Since(start), cmd.Err())
	return cmd
}

func (r *InstrumentedRedis) SAdd(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	start := time.Now()
	cmd := r.Cmdable.SAdd(ctx, key, members...)
	r.observeOperation("SADD", time.Since(start), cmd.Err())
	return cmd
}

func (r *InstrumentedRedis) SRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	start := time.Now()
	cmd := r.Cmdable.SRem(ctx, key, members...)
	r.observeOperation("SREM", time.Since(start), cmd.Err())
	return cmd
}

func (r *InstrumentedRedis) SIsMember(ctx context.Context, key string, member interface{}) *redis.BoolCmd {
	start := time.Now()
	cmd := r.Cmdable.SIsMember(ctx, key, member)
	r.observeOperation("SISMEMBER", time.Since(start), cmd.Err())
	return cmd
}

func (r *InstrumentedRedis) SCard(ctx context.Context, key string) *redis.IntCmd {
	start := time.Now()
	cmd := r.Cmdable.SCard(ctx, key)
	r.observeOperation("SCARD", time.Since(start), cmd.Err())
	return cmd
}

func (r *InstrumentedRedis) TTL(ctx context.Context, key string) *redis.DurationCmd {
	start := time.Now()
	cmd := r.Cmdable.TTL(ctx, key)
	r.observeOperation("TTL", time.Since(start), cmd.Err())
	return cmd
}

func (r *InstrumentedRedis) Type(ctx context.Context, key string) *redis.StatusCmd {
	start := time.Now()
	cmd := r.Cmdable.Type(ctx, key)
	r.observeOperation("TYPE", time.Since(start), cmd.Err())
	return cmd
}

func (r *InstrumentedRedis) Keys(ctx context.Context, pattern string) *redis.StringSliceCmd {
	start := time.Now()
	cmd := r.Cmdable.Keys(ctx, pattern)
	r.observeOperation("KEYS", time.Since(start), cmd.Err())
	return cmd
}

func (r *InstrumentedRedis) Ping(ctx context.Context) *redis.StatusCmd {
	start := time.Now()
	cmd := r.Cmdable.Ping(ctx)
	r.observeOperation("PING", time.Since(start), cmd.Err())
	return cmd
}

// Pipeline 返回的 pipeline 暂时直接透传, 不会对 pipeline 内部的命令做埋点。
// 已知限制: 使用 Pipeline 的链路(批量 MSET/MGET 等)目前不会出现在 ccnubox_redis_* 指标里。
// TODO: 实现一个 instrumentedPipeliner, 在 Exec() 时统一上报整体耗时和错误数。
func (r *InstrumentedRedis) Pipeline() redis.Pipeliner {
	return r.Cmdable.Pipeline()
}

// TxPipeline 同 Pipeline, 暂未埋点。详见 Pipeline 的注释。
func (r *InstrumentedRedis) TxPipeline() redis.Pipeliner {
	return r.Cmdable.TxPipeline()
}

var _ redis.Cmdable = (*InstrumentedRedis)(nil)
