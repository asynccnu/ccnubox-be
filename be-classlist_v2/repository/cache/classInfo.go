package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/be-classlist_v2/conf"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/repository/model"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/redis/go-redis/v9"
)

const RedisNull = "redis_Null"

type ClassInfoCache struct {
	BaseCache
	classExpiration     time.Duration
	blackListExpiration time.Duration
	log                 logger.Logger
}

func NewClassInfoCache(base BaseCache, cf *conf.ServerConf, l logger.Logger) *ClassInfoCache {
	classExpire := 24 * time.Hour
	if cf.ClassListConf.ClassExpiration > 0 {
		classExpire = time.Duration(cf.ClassListConf.ClassExpiration) * time.Millisecond
	}
	blackListExpiration := 1 * time.Minute
	if cf.ClassListConf.BlackListExpiration > 0 {
		blackListExpiration = time.Duration(cf.ClassListConf.BlackListExpiration) * time.Millisecond
	}
	return &ClassInfoCache{
		BaseCache:           base,
		classExpiration:     classExpire,
		blackListExpiration: blackListExpiration,
		log:                 l,
	}
}

func (c ClassInfoCache) generateClassInfosKey(stuId, xnm, xqm string) string {
	return fmt.Sprintf("ClassInfos:%s:%s:%s", stuId, xnm, xqm)
}

// GetClassInfosFromCache 从缓存中获取课程信息
func (c ClassInfoCache) GetClassInfosFromCache(ctx context.Context, stuId, xnm, xqm string) ([]*model.ClassInfo, error) {
	key := c.generateClassInfosKey(stuId, xnm, xqm)

	classInfos := make([]*model.ClassInfo, 0)
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, errorx.Errorf("cache.classInfo.GetClassInfosFromCache: key=%s miss: %w", key, err)
		}
		return nil, errorx.Errorf("cache.classInfo.GetClassInfosFromCache: key=%s: %w", key, err)
	}
	if val == RedisNull {
		return nil, nil
	}
	if err := json.Unmarshal([]byte(val), &classInfos); err != nil {
		return nil, errorx.Errorf("cache.classInfo.GetClassInfosFromCache: unmarshal key=%s, val=%q: %w", key, val, err)
	}
	return classInfos, nil
}

// AddClaInfosToCache 将整个课表转换成json格式，然后存到缓存中去
func (c ClassInfoCache) AddClaInfosToCache(ctx context.Context, stuId, xnm, xqm string, classInfos []*model.ClassInfo) error {
	key := c.generateClassInfosKey(stuId, xnm, xqm)
	var (
		val    string
		expire time.Duration
	)
	// 根据是否为空指针，来决定过期时间
	if classInfos == nil {
		val = RedisNull
		expire = c.blackListExpiration
	} else {
		valByte, err := json.Marshal(classInfos)
		if err != nil {
			return errorx.Errorf("cache.classInfo.AddClaInfosToCache: marshal key=%s, count=%d: %w", key, len(classInfos), err)
		}
		val = string(valByte)
		expire = c.classExpiration
	}

	if err := c.rdb.Set(ctx, key, val, expire).Err(); err != nil {
		return errorx.Errorf("cache.classInfo.AddClaInfosToCache: set key=%s: %w", key, err)
	}
	return nil
}

// DeleteClassInfoFromCache 删除课程信息缓存
func (c ClassInfoCache) DeleteClassInfoFromCache(ctx context.Context, stuId, xnm, xqm string) error {
	key := c.generateClassInfosKey(stuId, xnm, xqm)
	if err := c.rdb.Del(ctx, key).Err(); err != nil {
		return errorx.Errorf("cache.classInfo.DeleteClassInfoFromCache: key=%s: %w", key, err)
	}
	return nil
}
