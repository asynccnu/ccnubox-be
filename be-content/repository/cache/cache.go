package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/asynccnu/ccnubox-be/be-content/repository/model"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/redis/go-redis/v9"
)

// Cache 接口定义，包含获取、设置和清除数据的缓存方法
type Cache[T model.Content] interface {
	GetContent(ctx context.Context) ([]T, error)
	SetContent(ctx context.Context, val []T, expiration time.Duration) error
	ClearContent(ctx context.Context) error
}

type RedisCache[T model.Content] struct {
	cmd redis.Cmdable
}

// NewRedisCache 创建泛型缓存实例
func NewRedisCache[T model.Content](cmd redis.Cmdable) Cache[T] {
	return &RedisCache[T]{
		cmd: cmd,
	}
}

func (cache *RedisCache[T]) GetContent(ctx context.Context) ([]T, error) {
	var result []T
	key := cache.getRedisKey()

	data, err := cache.cmd.Get(ctx, key).Bytes()
	if err != nil {
		return result, errorx.Errorf("从缓存获取失败:%w", err)
	}

	err = json.Unmarshal(data, &result)
	if err != nil {
		return result, errorx.Errorf("json解析失败:%w", err)
	}

	return result, nil
}

func (cache *RedisCache[T]) SetContent(ctx context.Context, val []T, expiration time.Duration) error {
	key := cache.getRedisKey()
	data, err := json.Marshal(val)
	if err != nil {
		return errorx.Errorf("json解析失败:%w", err)
	}

	if err := cache.cmd.Set(ctx, key, data, expiration).Err(); err != nil {
		return errorx.Errorf("设置缓存失败:%w,val : %v", err, val)
	}

	return nil
}

func (cache *RedisCache[T]) ClearContent(ctx context.Context) error {
	key := cache.getRedisKey()
	if err := cache.cmd.Del(ctx, key).Err(); err != nil {
		return errorx.Errorf("清除缓存失败: %w, key: %s", err, key)
	}
	return nil
}

func (cache *RedisCache[T]) getRedisKey() string {
	var t T
	return "ccnubox:" + t.Type()
}
