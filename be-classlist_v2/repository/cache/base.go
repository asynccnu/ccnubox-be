package cache

import "github.com/redis/go-redis/v9"

type BaseCache struct {
	rdb redis.Cmdable
}

func NewBaseCache(rdb redis.Cmdable) BaseCache {
	return BaseCache{rdb: rdb}
}
