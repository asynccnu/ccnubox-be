package cache

import (
	"context"
	"github.com/redis/go-redis/v9"
	"strconv"
	"time"
)

const REDISKEY = "FUC:"

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
	return cache.cmd.Get(ctx, key).Int64()
}

func (cache *RedisCounterCache) SetCounterByStudentId(ctx context.Context, StudentId string, count int64) error {
	key := cache.getKey(StudentId)
	expiration := time.Hour * 24 * 7 // 一周的过期时间
	return cache.cmd.Set(ctx, key, count, expiration).Err()
}

// 获取所有 Counter
func (cache *RedisCounterCache) GetAllCounter(ctx context.Context) (Counters []*Counter, err error) {
	var cursor uint64
	var keys []string
	var countPerScan int64 = 100 // 每次 scan 获取的键数量

	for {
		// 使用 SCAN 获取一批键
		keys, cursor, err = cache.cmd.Scan(ctx, cursor, REDISKEY+"*", countPerScan).Result()
		if err != nil {
			return nil, err
		}

		if len(keys) > 0 {
			// 使用 MGET 一次性获取多个键的值
			values, err := cache.cmd.MGet(ctx, keys...).Result()
			if err != nil {
				return nil, err
			}

			// 遍历获取的键值对，将其转换为 Counter 对象
			for i, value := range values {
				if value != nil {
					StudentId := keys[i][4:]

					// 将值转换为 int64
					count, err := strconv.ParseInt(value.(string), 10, 64)
					if err != nil {
						return nil, err
					}

					Counters = append(Counters, &Counter{
						StudentId: StudentId,
						Count:     count,
					})
				}
			}
		}

		// 如果 cursor 为 0，表示扫描完成
		if cursor == 0 {
			break
		}
	}

	return Counters, nil
}

// 删除所有计数为 0 的 Counter
func (cache *RedisCounterCache) CleanZeroCounter(ctx context.Context) error {
	var cursor uint64
	var countPerScan int64 = 100 // 每次 scan 获取的键数量

	for {
		// 使用 SCAN 获取一批键
		keys, cursor, err := cache.cmd.Scan(ctx, cursor, REDISKEY+"*", countPerScan).Result()
		if err != nil {
			return err
		}

		if len(keys) > 0 {
			// 使用 MGET 一次性获取多个键的值
			values, err := cache.cmd.MGet(ctx, keys...).Result()
			if err != nil {
				return err
			}

			// 遍历获取的键值对，将其转换为 Counter 对象
			for i, value := range values {
				if value != nil {
					// 将值转换为 int64
					count, err := strconv.ParseInt(value.(string), 10, 64)
					if err != nil {
						return err
					}

					if count == 0 {
						err := cache.cmd.Del(ctx, keys[i]).Err()
						if err != nil {
							return err
						}
					}
				}
			}
		}

		// 如果 cursor 为 0，表示扫描完成
		if cursor == 0 {
			break
		}
	}
	return nil
}

// 批量设置多个 Counter
func (cache *RedisCounterCache) SetCounters(ctx context.Context, Counters []*Counter) error {
	pipe := cache.cmd.Pipeline()     // 使用 Pipeline 批量执行命令
	expiration := time.Hour * 24 * 7 // 设置每个键的过期时间为一周

	for _, Counter := range Counters {
		key := cache.getKey(Counter.StudentId)
		pipe.Set(ctx, key, Counter.Count, expiration) // 批量设置 Counter
	}

	_, err := pipe.Exec(ctx) // 执行批量命令
	if err != nil {
		return err
	}

	return nil
}

// 批量获取多个 Counter
func (cache *RedisCounterCache) GetCounters(ctx context.Context, StudentIds []string) (Counters []*Counter, err error) {
	// 构造 Redis 键
	keys := make([]string, len(StudentIds))
	for i, StudentId := range StudentIds {
		keys[i] = cache.getKey(StudentId)
	}
	// 使用 MGET 获取多个键对应的值
	values, err := cache.cmd.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	// 遍历获取的值，并转换为 Counter 对象
	for i, value := range values {
		if value != nil {
			count, err := strconv.ParseInt(value.(string), 10, 64)
			if err != nil {
				return nil, err
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
	// fuc的意思是Counter,这里减少键的长度来降低存储的键占用的内存
	return REDISKEY + StudentId
}

type Counter struct {
	StudentId string `json:"StudentId"`
	Count     int64  `json:"count"`
}
