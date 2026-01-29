package ioc

import (
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
)

// TODO 这里需要做适配，其他的redis部分都是直接返回一个cmd,这里则是一个client
func InitRedisLock(client *redis.Client) *redsync.Redsync {
	pool := goredis.NewPool(client)
	rs := redsync.New(pool)
	return rs
}
