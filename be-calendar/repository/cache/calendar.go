package cache

import (
	"context"
	"encoding/json"
	"github.com/asynccnu/ccnubox-be/be-calendar/domain" // 替换为calendar的domain路径
	"github.com/redis/go-redis/v9"
	"strconv"
)

// CalendarCache 接口定义，包含获取、设置和清除日历数据的缓存方法
type CalendarCache interface {
	GetCalendar(ctx context.Context, year int64) (*domain.Calendar, error)
	SetCalendar(ctx context.Context, calendars *domain.Calendar, year int64) error
	ClearCalendarCache(ctx context.Context, year int64) error // 添加清除缓存的方法
}

// RedisCalendarCache 结构体，实现了 CalendarCache 接口
type RedisCalendarCache struct {
	cmd redis.Cmdable
}

// NewRedisCalendarCache 创建一个基于 Redis 的 CalendarCache 实现
func NewRedisCalendarCache(cmd redis.Cmdable) CalendarCache {
	return &RedisCalendarCache{cmd: cmd}
}

// GetCalendar 从缓存中获取日历数据
func (cache *RedisCalendarCache) GetCalendar(ctx context.Context, year int64) (*domain.Calendar, error) {
	key := cache.getKey(year) // 获取缓存的键

	data, err := cache.cmd.Get(ctx, key).Bytes() // 从Redis中获取数据,这里出现panic
	if err != nil {
		return &domain.Calendar{}, err
	}
	var st domain.Calendar
	err = json.Unmarshal(data, &st) // 反序列化数据
	return &st, err
}

// SetCalendar 将日历数据存储到缓存中
func (cache *RedisCalendarCache) SetCalendar(ctx context.Context, calendars *domain.Calendar, year int64) error {
	key := cache.getKey(year)             // 获取缓存的键
	data, err := json.Marshal(*calendars) // 序列化数据
	if err != nil {
		return err
	}
	return cache.cmd.Set(ctx, key, data, 0).Err() // 永不过期
}

// ClearCalendarCache 清除指定年份的日历数据缓存
func (cache *RedisCalendarCache) ClearCalendarCache(ctx context.Context, year int64) error {
	key := cache.getKey(year)            // 获取缓存的键
	return cache.cmd.Del(ctx, key).Err() // 从Redis中删除该键
}

// getKey 返回缓存键名
func (cache *RedisCalendarCache) getKey(key int64) string {
	return "ccnubox:calendars:" + strconv.FormatInt(key, 10)
}
