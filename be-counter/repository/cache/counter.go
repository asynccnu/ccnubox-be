package cache

import (
	"context"
	"strconv"

	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/redis/go-redis/v9"
)

const REDISKEY = "ccnubox:FUC"

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

	val, err := cache.cmd.HGet(ctx, REDISKEY, StudentId).Int64()
	if err != nil {
		if errorx.Is(err, redis.Nil) {
			return 0, nil
		}
		return 0, errorx.Errorf("cache: get counter failed, studentId: %s, err: %w", StudentId, err)
	}
	return val, nil
}

func (cache *RedisCounterCache) SetCounterByStudentId(ctx context.Context, StudentId string, count int64) error {
	err := cache.cmd.HSet(ctx, REDISKEY, StudentId, count).Err()
	if err != nil {
		return errorx.Errorf("cache: set counter failed, studentId: %s, err: %w", StudentId, err)
	}
	return nil
}

// 获取所有 Counter
func (cache *RedisCounterCache) GetAllCounter(ctx context.Context) (Counters []*Counter, err error) {
	var cursor uint64
	var result []string
	for {
		result, cursor, err = cache.cmd.HScan(ctx, REDISKEY, cursor, "*", 500).Result()
		if err != nil {
			return nil, errorx.Errorf("cache: HGetAll keys failed: %w", err)
		}

		for i := 0; i < len(result); i += 2 {
			id := result[i]
			key := result[i+1]
			cnt, err := strconv.ParseInt(key, 10, 64)
			if err != nil {
				return nil, errorx.Errorf("cache: get all parse failed, key: %s, err: %w", id, err)
			}
			Counters = append(Counters, &Counter{StudentId: id, Count: cnt})
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
	var result []string
	var err error
	var delFields []string
	for {
		result, cursor, err = cache.cmd.HScan(ctx, REDISKEY, cursor, "*", countPerScan).Result()
		if err != nil {
			return errorx.Errorf("cache: clean scan failed: %w", err)
		}

		for i := 0; i < len(result); i += 2 {
			key := result[i]
			val := result[i+1]
			cnt, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return errorx.Errorf("cache: get clean parse failed, key: %s, err: %w", key, err)
			}
			if cnt == 0 {
				delFields = append(delFields, key)
			}
		}

		if cursor == 0 {
			break
		}
	}

	if len(delFields) > 0 {
		_, err := cache.cmd.HDel(ctx, REDISKEY, delFields...).Result()
		if err != nil {
			return errorx.Errorf("cache: del fields failed,err:%w", err)
		}
	}
	return nil
}

// 批量设置多个 Counter
func (cache *RedisCounterCache) SetCounters(ctx context.Context, Counters []*Counter) error {
	pipe := cache.cmd.Pipeline()

	for _, c := range Counters {
		pipe.HSet(ctx, c.StudentId, REDISKEY, c.Count)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return errorx.Errorf("cache: pipeline set counters failed: %w", err)
	}

	return nil
}

// 批量获取多个 Counter
func (cache *RedisCounterCache) GetCounters(ctx context.Context, StudentIds []string) (Counters []*Counter, err error) {
	fields := make([]string, len(StudentIds))
	for i, sid := range StudentIds {
		fields[i] = sid
	}

	values, err := cache.cmd.HMGet(ctx, REDISKEY, fields...).Result()
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

type Counter struct {
	StudentId string `json:"StudentId"`
	Count     int64  `json:"count"`
}
