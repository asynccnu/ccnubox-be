package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/tool"
	"github.com/redis/go-redis/v9"
)

var (
	// ErrKeyNotFound 定义为 redis.Nil 的别名，方便 Service 层做语义判断
	ErrKeyNotFound = redis.Nil
)

type UserCache interface {
	GetCookie(ctx context.Context, sid string, system tool.XKSystem) (string, error)
	SetCookie(ctx context.Context, sid string, cookie string, system tool.XKSystem) error
	GetLibraryToken(ctx context.Context, sid string, serviceType string) (string, error)
	SetLibraryToken(ctx context.Context, sid string, cookie string, serviceType string) error
	DeleteUserData(ctx context.Context, sid string) error
}

type RedisUserCache struct {
	cmd redis.Cmdable
}

// NewRedisUserCache 创建一个新的 RedisUserCache 实例
func NewRedisUserCache(cmd redis.Cmdable) UserCache {
	return &RedisUserCache{cmd: cmd}
}

// GetCookie 从 Redis 获取指定 sid 对应的教务系统 cookie
func (cache *RedisUserCache) GetCookie(ctx context.Context, sid string, system tool.XKSystem) (string, error) {
	key := cache.cookieKey(sid, system)
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
func (cache *RedisUserCache) SetCookie(ctx context.Context, sid string, cookie string, system tool.XKSystem) error {
	key := cache.cookieKey(sid, system)
	// 过期时间设为 5 分钟，教务系统 Session 较短，不宜设置过长
	err := cache.cmd.Set(ctx, key, cookie, 5*time.Minute).Err()
	if err != nil {
		return errorx.Errorf("cache: redis set cookie failed, key: %s, err: %w", key, err)
	}
	return nil
}

// GetLibraryCookie 从 Redis 获取指定 sid 对应的图书馆 cookie
func (cache *RedisUserCache) GetLibraryToken(ctx context.Context, sid string, serviceType string) (string, error) {
	key := cache.libraryKey(sid, serviceType)
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
func (cache *RedisUserCache) SetLibraryToken(ctx context.Context, sid string, cookie string, serviceType string) error {
	key := cache.libraryKey(sid, serviceType)
	err := cache.cmd.Set(ctx, key, cookie, 5*time.Minute).Err()
	if err != nil {
		return errorx.Errorf("cache: redis set library cookie failed, key: %s, err: %w", key, err)
	}
	return nil
}

func (cache *RedisUserCache) DeleteUserData(ctx context.Context, sid string) error {
	keys := []string{
		cache.legacyCookieKey(sid), // 迁移期兼容删除旧 key，正常读写仅使用带系统类型的新 key。
		cache.cookieKey(sid, tool.XKUndergraduate),
		cache.cookieKey(sid, tool.XKPostgraduate),
		cache.libraryKey(sid, "LIBRARY_SEAT"),
		cache.libraryKey(sid, "LIBRARY_DISCUSSION"),
	}
	if err := cache.cmd.Del(ctx, keys...).Err(); err != nil {
		return errorx.Errorf("cache: delete user data failed, sid: %s, err: %w", sid, err)
	}
	return nil
}

func (cache *RedisUserCache) legacyCookieKey(sid string) string {
	return fmt.Sprintf("ccnubox:users:xk:%s", sid)
}

func (cache *RedisUserCache) cookieKey(sid string, system tool.XKSystem) string {
	return fmt.Sprintf("ccnubox:users:xk:v2:%s:%s", system, sid)
}

func (cache *RedisUserCache) libraryKey(sid string, service string) string {
	return fmt.Sprintf("ccnubox:users:lib:%s:%s", service, sid) // 统一命名层级
}
