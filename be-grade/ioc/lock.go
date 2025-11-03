package ioc

import (
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
)

func InitRedisLock(client *redis.Client) *redsync.Redsync {
	pool := goredis.NewPool(client)
	rs := redsync.New(pool)

	return rs
}
