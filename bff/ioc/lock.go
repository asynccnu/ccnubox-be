package ioc

import (
	"fmt"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
)

func InitRedisLock(client redis.Cmdable) *redsync.Redsync {
	cli, ok := client.(*redis.Client)
	if !ok {
		panic(fmt.Errorf("init redis lock error"))
	}
	pool := goredis.NewPool(cli)
	rs := redsync.New(pool)
	return rs
}
