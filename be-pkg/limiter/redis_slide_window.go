package limiter

import (
	"context"
	_ "embed"
	"github.com/redis/go-redis/v9"
	"time"
)

//go:embed slide_window.lua
var luaScript string

type RedisSlideWindowLimiter struct {
	cmd        redis.Cmdable
	interval   time.Duration
	thresholds int // 阈值
}

func NewRedisSlideWindowLimiter(cmd redis.Cmdable, interval time.Duration, thresholds int) Limiter {
	return &RedisSlideWindowLimiter{
		cmd:        cmd,
		interval:   interval,
		thresholds: thresholds,
	}
}

func (r *RedisSlideWindowLimiter) Limit(ctx context.Context, key string) (bool, error) {
	return r.cmd.Eval(ctx, luaScript, []string{key}, r.interval.Milliseconds(), r.thresholds, time.Now().UnixMilli()).Bool()
}
