package lock

import (
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
)

// redis 分布式锁

type RedisLocker struct {
	mu *redsync.Mutex
}

func (rl *RedisLocker) Lock() error {
	return rl.mu.Lock()
}
func (rl *RedisLocker) Unlock() (bool, error) {
	return rl.mu.Unlock()
}

type RedisLockBuilder struct {
	rs *redsync.Redsync
}

func NewRedisLockBuilder(client *redis.Client) Builder {
	pool := goredis.NewPool(client)
	rs := redsync.New(pool)
	return &RedisLockBuilder{
		rs: rs,
	}
}

func (rlb *RedisLockBuilder) Build(name string) Locker {
	return &RedisLocker{
		mu: rlb.rs.NewMutex(name, redsync.WithTries(1)),
	}
}

// 目前是加载缓存空教室用，全程加载时间比较长
func (rlb *RedisLockBuilder) BuildWithExpire(name string, expire time.Duration) Locker {
	return &RedisLocker{
		mu: rlb.rs.NewMutex(name, redsync.WithTries(1), redsync.WithExpiry(expire)),
	}
}
