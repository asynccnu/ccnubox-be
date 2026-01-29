package ioc

import (
	"context"
	"fmt"

	"github.com/asynccnu/ccnubox-be/be-grade/conf"
	"github.com/go-kratos/kratos/v2/log"

	"github.com/redis/go-redis/v9"
)

func InitRedisClient(cfg *conf.InfraConf) *redis.Client {
	cmd := redis.NewClient(&redis.Options{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password})

	if err := cmd.Ping(context.Background()).Err(); err != nil {
		log.Fatal(fmt.Sprintf("Redis 连接失败: %v", err))
	}
	return cmd
}
