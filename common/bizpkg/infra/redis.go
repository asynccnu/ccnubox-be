package infra

import (
	"context"
	"fmt"

	"github.com/asynccnu/ccnubox-be/common/bizpkg/conf"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
)

func InitRedis(cfg *conf.RedisConf) *redis.Client {
	cmd := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		// 让命令读写 socket 时使用调用方 context 的 deadline；
		// 否则即使传入带超时的 context，go-redis 也只用客户端 ReadTimeout/WriteTimeout。
		ContextTimeoutEnabled: true,
	})

	if err := cmd.Ping(context.Background()).Err(); err != nil {
		log.Fatal(fmt.Sprintf("Redis 连接失败: %v", err))
	}

	return cmd
}
