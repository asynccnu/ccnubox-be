package cache

import (
	"context"
	"encoding/json"
	"github.com/asynccnu/ccnubox-be/be-banner/domain"
	"github.com/redis/go-redis/v9"
)

type BannerCache interface {
	GetBanners(ctx context.Context) ([]*domain.Banner, error)
	SetBanners(ctx context.Context, banners []*domain.Banner) error
}

type RedisBannerCache struct {
	cmd redis.Cmdable
}

func NewRedisBannerCache(cmd redis.Cmdable) BannerCache {
	return &RedisBannerCache{cmd: cmd}
}

func (cache *RedisBannerCache) GetBanners(ctx context.Context) ([]*domain.Banner, error) {
	key := cache.getKey()
	data, err := cache.cmd.Get(ctx, key).Bytes()
	if err != nil {
		return []*domain.Banner{}, err
	}

	var st []*domain.Banner
	err = json.Unmarshal(data, &st)
	return st, err
}

func (cache *RedisBannerCache) SetBanners(ctx context.Context, banners []*domain.Banner) error {
	key := cache.getKey()
	data, err := json.Marshal(banners)
	if err != nil {
		return err
	}
	return cache.cmd.Set(ctx, key, data, 0).Err() // 永不过期
}

func (cache *RedisBannerCache) getKey() string {
	return "banners"
}
