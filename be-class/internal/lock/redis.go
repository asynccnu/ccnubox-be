package lock

import (
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
