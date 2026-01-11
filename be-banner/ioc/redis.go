package ioc

import (
	"context"
	"fmt"

	"github.com/asynccnu/ccnubox-be/be-banner/conf"
	"github.com/redis/go-redis/v9"
)

func InitRedis(cfg *conf.InfraConf) redis.Cmdable {
	cmd := redis.NewClient(&redis.Options{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password})

	ctx := context.Background()
	if err := cmd.Ping(ctx).Err(); err != nil {
		panic(fmt.Sprintf("Redis 连接失败: %v", err))
	}

	return cmd
}
