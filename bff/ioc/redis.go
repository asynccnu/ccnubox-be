package ioc

import (
	"github.com/asynccnu/ccnubox-be/bff/conf"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/infra"
	"github.com/redis/go-redis/v9"
)

func InitRedis(cfg *conf.InfraConf) *redis.Client {
	return infra.InitRedis(cfg.Redis)
}

func RedisCmdable(client *redis.Client) redis.Cmdable {
	return client
}
