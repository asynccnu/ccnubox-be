package cache

import "github.com/redis/go-redis/v9"

type BaseCache struct {
	rdb *redis.Client
}

func NewBaseCache(rdb *redis.Client) BaseCache {
	return BaseCache{rdb: rdb}
}
