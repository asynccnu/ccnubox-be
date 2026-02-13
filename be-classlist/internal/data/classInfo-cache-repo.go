package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/be-classlist/internal/conf"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/data/do"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/redis/go-redis/v9"
)

const RedisNull = "redis_Null"

type ClassInfoCacheRepo struct {
	rdb                 *redis.Client
	classExpiration     time.Duration
	blackListExpiration time.Duration
}

func NewClassInfoCacheRepo(rdb *redis.Client, cf *conf.Server) *ClassInfoCacheRepo {
	classExpire := 5 * 24 * time.Hour
	if cf.ClassExpiration > 0 {
		classExpire = time.Duration(cf.ClassExpiration) * time.Second
	}
	blackListExpiration := 1 * time.Minute
	if cf.BlackListExpiration > 0 {
		blackListExpiration = time.Duration(cf.BlackListExpiration) * time.Second
	}
	return &ClassInfoCacheRepo{
		rdb:                 rdb,
		classExpiration:     classExpire,
		blackListExpiration: blackListExpiration,
	}
}

// AddClaInfosToCache 将整个课表转换成json格式，然后存到缓存中去
func (c ClassInfoCacheRepo) AddClaInfosToCache(ctx context.Context, key string, classInfos []*do.ClassInfo) error {
	var (
		val    string
		expire time.Duration
		// 根据是否为空指针，来决定过期时间
	)
	logh := logger.GetLoggerFromCtx(ctx)
	// 检查classInfos是否为空指针
	if classInfos == nil {
		val = RedisNull
		expire = c.blackListExpiration
	} else {
		valByte, err := json.Marshal(classInfos)
		if err != nil {
			logh.Errorf("json Marshal (%v) err: %v", classInfos, err)
			return err
		}
		val = string(valByte)
		expire = c.classExpiration
	}

	err := c.rdb.Set(ctx, key, val, expire).Err()
	if err != nil {
		logh.Errorf("Redis:Set k(%s)-v(%s) failed: %v", key, val, err)
		return err
	}
	return nil
}

func (c ClassInfoCacheRepo) GetClassInfosFromCache(ctx context.Context, key string) ([]*do.ClassInfo, error) {
	logh := logger.GetLoggerFromCtx(ctx)

	classInfos := make([]*do.ClassInfo, 0)
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, fmt.Errorf("error getting classlist info from cache: %w", err)
		}
		logh.Errorf("Redis:get key(%s) failed: %v", key, err)
		return nil, err
	}
	if val == RedisNull {
		return nil, nil
	}
	err = json.Unmarshal([]byte(val), &classInfos)
	if err != nil {
		logh.Errorf("json Unmarshal (%v) failed: %v", val, err)
		return nil, err
	}
	return classInfos, nil
}

func (c ClassInfoCacheRepo) DeleteClassInfoFromCache(ctx context.Context, classInfosKey ...string) error {
	logh := logger.GetLoggerFromCtx(ctx)
	if err := c.rdb.Del(ctx, classInfosKey...).Err(); err != nil {
		logh.Errorf("redis delete key{%v} failed: %v", classInfosKey, err)
	}
	return nil
}
