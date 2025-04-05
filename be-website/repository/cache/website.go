package cache

import (
	"context"
	"encoding/json"
	"github.com/asynccnu/ccnubox-be/be-website/domain"
	"github.com/redis/go-redis/v9"
)

type WebsiteCache interface {
	GetWebsites(ctx context.Context) ([]*domain.Website, error)
	SetWebsites(ctx context.Context, websites []*domain.Website) error
}

type RedisWebsiteCache struct {
	cmd redis.Cmdable
}

func NewRedisWebsiteCache(cmd redis.Cmdable) WebsiteCache {
	return &RedisWebsiteCache{cmd: cmd}
}

func (cache *RedisWebsiteCache) GetWebsites(ctx context.Context) ([]*domain.Website, error) {
	key := cache.getKey()
	data, err := cache.cmd.Get(ctx, key).Bytes()
	if err != nil {
		return []*domain.Website{}, err
	}
	var st []*domain.Website
	err = json.Unmarshal(data, &st)
	return st, err
}

func (cache *RedisWebsiteCache) SetWebsites(ctx context.Context, websites []*domain.Website) error {
	key := cache.getKey()
	data, err := json.Marshal(websites)
	if err != nil {
		return err
	}
	return cache.cmd.Set(ctx, key, data, 0).Err() // 永不过期
}

func (cache *RedisWebsiteCache) getKey() string {
	return "websites"
}
