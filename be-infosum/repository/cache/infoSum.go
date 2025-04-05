package cache

import (
	"context"
	"encoding/json"
	"github.com/asynccnu/ccnubox-be/be-infosum/domain"
	"github.com/redis/go-redis/v9"
)

type InfoSumCache interface {
	GetInfoSums(ctx context.Context) ([]*domain.InfoSum, error)
	SetInfoSums(ctx context.Context, InfoSums []*domain.InfoSum) error
}

type RedisInfoSumCache struct {
	cmd redis.Cmdable
}

func NewRedisInfoSumCache(cmd redis.Cmdable) InfoSumCache {
	return &RedisInfoSumCache{cmd: cmd}
}

func (cache *RedisInfoSumCache) GetInfoSums(ctx context.Context) ([]*domain.InfoSum, error) {
	key := cache.getKey()
	data, err := cache.cmd.Get(ctx, key).Bytes()
	if err != nil {
		return []*domain.InfoSum{}, err
	}
	var st []*domain.InfoSum
	err = json.Unmarshal(data, &st)
	return st, err
}

func (cache *RedisInfoSumCache) SetInfoSums(ctx context.Context, InfoSums []*domain.InfoSum) error {
	key := cache.getKey()
	data, err := json.Marshal(InfoSums)
	if err != nil {
		return err
	}
	return cache.cmd.Set(ctx, key, data, 0).Err() // 永不过期
}

func (cache *RedisInfoSumCache) getKey() string {
	return "InfoSums"
}
