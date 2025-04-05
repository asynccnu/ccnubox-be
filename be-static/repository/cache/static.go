package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/asynccnu/ccnubox-be/be-static/domain"
	"github.com/redis/go-redis/v9"
	"time"
)

type StaticCache interface {
	GetStatic(ctx context.Context, name string) (domain.Static, error)
	SetStatic(ctx context.Context, static domain.Static) error
}

type RedisStaticCache struct {
	cmd redis.Cmdable
}

func NewRedisStaticCache(cmd redis.Cmdable) StaticCache {
	return &RedisStaticCache{cmd: cmd}
}

func (cache *RedisStaticCache) GetStatic(ctx context.Context, name string) (domain.Static, error) {
	key := cache.staticKey(name)
	data, err := cache.cmd.Get(ctx, key).Bytes()
	if err != nil {
		return domain.Static{}, err
	}
	var st domain.Static
	err = json.Unmarshal(data, &st)
	return st, err
}

func (cache *RedisStaticCache) SetStatic(ctx context.Context, static domain.Static) error {
	key := cache.staticKey(static.Name)
	data, err := json.Marshal(static)
	if err != nil {
		return err
	}
	return cache.cmd.Set(ctx, key, data, time.Hour*24*7).Err()
}

func (cache *RedisStaticCache) staticKey(name string) string {
	return fmt.Sprintf("ccnubox:static:%s", name)
}
