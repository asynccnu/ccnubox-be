package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/conf"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/model"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redis/redis"
	"time"
)

const RedisNull = "redis_Null"

type ClassInfoCacheRepo struct {
	rdb                 *redis.Client
	log                 *log.Helper
	classExpiration     time.Duration
	blackListExpiration time.Duration
}

func NewClassInfoCacheRepo(rdb *redis.Client, cf *conf.Server, logger log.Logger) *ClassInfoCacheRepo {
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
		log:                 log.NewHelper(logger),
		classExpiration:     classExpire,
		blackListExpiration: blackListExpiration,
	}
}

// AddClaInfosToCache 将整个课表转换成json格式，然后存到缓存中去
func (c ClassInfoCacheRepo) AddClaInfosToCache(ctx context.Context, key string, classInfos []*model.ClassInfo) error {
	var (
		val    string
		expire time.Duration
		//根据是否为空指针，来决定过期时间
	)
	//检查classInfos是否为空指针
	if classInfos == nil {
		val = RedisNull
		expire = c.blackListExpiration
	} else {
		valByte, err := json.Marshal(classInfos)
		if err != nil {
			c.log.Errorf("json Marshal (%v) err: %v", classInfos, err)
			return err
		}
		val = string(valByte)
		expire = c.classExpiration
	}

	err := c.rdb.Set(key, val, expire).Err()
	if err != nil {
		c.log.Errorf("Redis:Set k(%s)-v(%s) failed: %v", key, val, err)
		return err
	}
	return nil
}
func (c ClassInfoCacheRepo) GetClassInfosFromCache(ctx context.Context, key string) ([]*model.ClassInfo, error) {
	var classInfos = make([]*model.ClassInfo, 0)
	val, err := c.rdb.Get(key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, fmt.Errorf("error getting classlist info from cache: %w", err)
		}
		c.log.Errorf("Redis:get key(%s) failed: %v", key, err)
		return nil, err
	}
	if val == RedisNull {
		return nil, nil
	}
	err = json.Unmarshal([]byte(val), &classInfos)
	if err != nil {
		c.log.Errorf("json Unmarshal (%v) failed: %v", val, err)
		return nil, err
	}
	return classInfos, nil
}

func (c ClassInfoCacheRepo) DeleteClassInfoFromCache(ctx context.Context, classInfosKey ...string) error {
	if err := c.rdb.Del(classInfosKey...).Err(); err != nil {
		c.log.Errorf("redis delete key{%v} failed: %v", classInfosKey, err)
	}
	return nil
}
