package ioc

import (
	"github.com/asynccnu/ccnubox-be/be-elecprice/conf"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/infra"
	"github.com/redis/go-redis/v9"
)

func InitRedis(cfg *conf.InfraConf) redis.Cmdable {
	return infra.InitRedis(cfg.Redis)
}
