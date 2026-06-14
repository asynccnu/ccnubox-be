package ioc

import (
	"github.com/asynccnu/ccnubox-be/bff/conf"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/infra"
	"github.com/asynccnu/ccnubox-be/common/pkg/metricsx"
	"github.com/redis/go-redis/v9"
)

func InitRedis(cfg *conf.InfraConf) *redis.Client {
	return infra.InitRedis(cfg.Redis)
}

// RedisCmdable 返回被 metrics 包装后的 Redis client，自动记录所有 Redis 操作的耗时和计数
func RedisCmdable(client *redis.Client, m *metricsx.Metrics) redis.Cmdable {
	return metricsx.NewInstrumentedRedis(client, m)
}
