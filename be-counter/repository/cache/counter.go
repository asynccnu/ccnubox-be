package cache

import (
	"context"
	"strconv"
	"time"

	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/redis/go-redis/v9"
)

const REDISKEY = "ccnubox:FUC:"

type CounterCache interface {
	GetCounterByStudentId(ctx context.Context, StudentId string) (count int64, err error)
	SetCounterByStudentId(ctx context.Context, StudentId string, count int64) error
	GetAllCounter(ctx context.Context) (Counters []*Counter, err error)
	GetCounters(ctx context.Context, StudentIds []string) (Counters []*Counter, err error)
	SetCounters(ctx context.Context, Counters []*Counter) error
	CleanZeroCounter(ctx context.Context) error
}

type RedisCounterCache struct {
	cmd redis.Cmdable
}

func NewRedisCounterCache(cmd redis.Cmdable) CounterCache {
	return &RedisCounterCache{cmd: cmd}
}

func (cache *RedisCounterCache) GetCounterByStudentId(ctx context.Context, StudentId string) (count int64, err error) {
	key := cache.getKey(StudentId)
	val, err := cache.cmd.Get(ctx, key).Int64()
	if err != nil {
		if errorx.Is(err, redis.Nil) {
			return 0, nil
		}
		return 0, errorx.Errorf("cache: get counter failed, studentId: %s, err: %w", StudentId, err)
	}
	return val, nil
}

func (cache *RedisCounterCache) SetCounterByStudentId(ctx context.Context, StudentId string, count int64) error {
	key := cache.getKey(StudentId)
	expiration := time.Hour * 24 * 7 // 一周的过期时间
	err := cache.cmd.Set(ctx, key, count, expiration).Err()
	if err != nil {
		return errorx.Errorf("cache: set counter failed, studentId: %s, err: %w", StudentId, err)
	}
	return nil
}

// 获取所有 Counter
func (cache *RedisCounterCache) GetAllCounter(ctx context.Context) (Counters []*Counter, err error) {
	var cursor uint64
	var keys []string
	var countPerScan int64 = 100

	for {
		keys, cursor, err = cache.cmd.Scan(ctx, cursor, REDISKEY+"*", countPerScan).Result()
		if err != nil {
			return nil, errorx.Errorf("cache: scan keys failed: %w", err)
		}

		if len(keys) > 0 {
			values, err := cache.cmd.MGet(ctx, keys...).Result()
			if err != nil {
				return nil, errorx.Errorf("cache: mget keys failed: %w", err)
			}

			for i, value := range values {
				if value != nil {
					// 假设前缀长度固定，这里可以做更健壮的处理
					studentId := keys[i][len(REDISKEY):]

					count, err := strconv.ParseInt(value.(string), 10, 64)
					if err != nil {
						return nil, errorx.Errorf("cache: parse count failed, key: %s, err: %w", keys[i], err)
					}

					Counters = append(Counters, &Counter{
						StudentId: studentId,
						Count:     count,
					})
				}
			}
		}

		if cursor == 0 {
			break
		}
	}

	return Counters, nil
}

// 删除所有计数为 0 的 Counter
func (cache *RedisCounterCache) CleanZeroCounter(ctx context.Context) error {
	var cursor uint64
	var countPerScan int64 = 100

	for {
		keys, cursor, err := cache.cmd.Scan(ctx, cursor, REDISKEY+"*", countPerScan).Result()
		if err != nil {
			return errorx.Errorf("cache: clean scan failed: %w", err)
		}

		if len(keys) > 0 {
			values, err := cache.cmd.MGet(ctx, keys...).Result()
			if err != nil {
				return errorx.Errorf("cache: clean mget failed: %w", err)
			}

			for i, value := range values {
				if value != nil {
					count, err := strconv.ParseInt(value.(string), 10, 64)
					if err != nil {
						return errorx.Errorf("cache: clean parse failed, key: %s, err: %w", keys[i], err)
					}

					if count == 0 {
						err := cache.cmd.Del(ctx, keys[i]).Err()
						if err != nil {
							return errorx.Errorf("cache: clean delete failed, key: %s, err: %w", keys[i], err)
						}
					}
				}
			}
		}

		if cursor == 0 {
			break
		}
	}
	return nil
}

// 批量设置多个 Counter
func (cache *RedisCounterCache) SetCounters(ctx context.Context, Counters []*Counter) error {
	pipe := cache.cmd.Pipeline()
	expiration := time.Hour * 24 * 7

	for _, c := range Counters {
		key := cache.getKey(c.StudentId)
		pipe.Set(ctx, key, c.Count, expiration)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return errorx.Errorf("cache: pipeline set counters failed: %w", err)
	}

	return nil
}

// 批量获取多个 Counter
func (cache *RedisCounterCache) GetCounters(ctx context.Context, StudentIds []string) (Counters []*Counter, err error) {
	keys := make([]string, len(StudentIds))
	for i, sid := range StudentIds {
		keys[i] = cache.getKey(sid)
	}

	values, err := cache.cmd.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, errorx.Errorf("cache: batch get counters failed: %w", err)
	}

	for i, value := range values {
		if value != nil {
			count, err := strconv.ParseInt(value.(string), 10, 64)
			if err != nil {
				return nil, errorx.Errorf("cache: parse batch count failed, studentId: %s, err: %w", StudentIds[i], err)
			}

			Counters = append(Counters, &Counter{
				StudentId: StudentIds[i],
				Count:     count,
			})
		}
	}

	return Counters, nil
}

func (cache *RedisCounterCache) getKey(StudentId string) string {
	return REDISKEY + StudentId
}

type Counter struct {
	StudentId string `json:"StudentId"`
	Count     int64  `json:"count"`
}
