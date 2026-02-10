package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/redis/go-redis/v9"
)

var (
	// ErrKeyNotFound 定义为 redis.Nil 的别名，方便 Service 层做语义判断
	ErrKeyNotFound = redis.Nil
)

type UserCache interface {
	GetCookie(ctx context.Context, sid string) (string, error)
	SetCookie(ctx context.Context, sid string, cookie string) error
	GetLibraryCookie(ctx context.Context, sid string) (string, error)
	SetLibraryCookie(ctx context.Context, sid string, cookie string) error
}

type RedisUserCache struct {
	cmd redis.Cmdable
}

// NewRedisUserCache 创建一个新的 RedisUserCache 实例
func NewRedisUserCache(cmd redis.Cmdable) UserCache {
	return &RedisUserCache{cmd: cmd}
}

// GetCookie 从 Redis 获取指定 sid 对应的教务系统 cookie
func (cache *RedisUserCache) GetCookie(ctx context.Context, sid string) (string, error) {
	key := cache.key(sid)
	val, err := cache.cmd.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", ErrKeyNotFound
		}
		return "", errorx.Errorf("cache: redis get cookie failed, key: %s, err: %w", key, err)
	}
	return val, nil
}

// SetCookie 将 sid 和对应的 cookie 存入 Redis，过期时间 5 分钟
func (cache *RedisUserCache) SetCookie(ctx context.Context, sid string, cookie string) error {
	key := cache.key(sid)
	// 过期时间设为 5 分钟，教务系统 Session 较短，不宜设置过长
	err := cache.cmd.Set(ctx, key, cookie, 5*time.Minute).Err()
	if err != nil {
		return errorx.Errorf("cache: redis set cookie failed, key: %s, err: %w", key, err)
	}
	return nil
}

// GetLibraryCookie 从 Redis 获取指定 sid 对应的图书馆 cookie
func (cache *RedisUserCache) GetLibraryCookie(ctx context.Context, sid string) (string, error) {
	key := cache.libraryKey(sid)
	val, err := cache.cmd.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", ErrKeyNotFound
		}
		return "", errorx.Errorf("cache: redis get library cookie failed, key: %s, err: %w", key, err)
	}
	return val, nil
}

// SetLibraryCookie 将 sid 和对应的图书馆 cookie 存入 Redis
func (cache *RedisUserCache) SetLibraryCookie(ctx context.Context, sid string, cookie string) error {
	key := cache.libraryKey(sid)
	err := cache.cmd.Set(ctx, key, cookie, 5*time.Minute).Err()
	if err != nil {
		return errorx.Errorf("cache: redis set library cookie failed, key: %s, err: %w", key, err)
	}
	return nil
}

func (cache *RedisUserCache) key(sid string) string {
	return fmt.Sprintf("ccnubox:users:xk:%s", sid) // 增加 xk 标识区分业务
}

func (cache *RedisUserCache) libraryKey(sid string) string {
	return fmt.Sprintf("ccnubox:users:lib:%s", sid) // 统一命名层级
}
