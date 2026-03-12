package ioc

import (
	"github.com/asynccnu/ccnubox-be/be-grade/conf"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/infra"
	"github.com/redis/go-redis/v9"
)

func InitRedisClient(cfg *conf.InfraConf) *redis.Client {
	return infra.InitRedis(cfg.Redis)
}
