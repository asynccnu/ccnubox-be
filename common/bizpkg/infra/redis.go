package infra

import (
	"context"
	"fmt"

	"github.com/asynccnu/ccnubox-be/common/bizpkg/conf"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
)

func InitRedis(cfg *conf.RedisConf) redis.Cmdable {
	cmd := redis.NewClient(&redis.Options{Addr: cfg.Addr, Password: cfg.Password})

	if err := cmd.Ping(context.Background()).Err(); err != nil {
		log.Fatal(fmt.Sprintf("Redis 连接失败: %v", err))
	}

	return cmd
}
