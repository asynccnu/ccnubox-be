package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/asynccnu/ccnubox-be/Muxi_ClassList/internal/classLog"
	"github.com/asynccnu/ccnubox-be/Muxi_ClassList/internal/model"
	"github.com/go-redis/redis"
	"time"
)

const RedisNull = "redis_Null"

type ClassInfoCacheRepo struct {
	rdb *redis.Client
	log classLog.Clogger
}

func NewClassInfoCacheRepo(rdb *redis.Client, logger classLog.Clogger) *ClassInfoCacheRepo {
	return &ClassInfoCacheRepo{
		rdb: rdb,
		log: logger,
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
		expire = BlackListExpiration
	} else {
		valByte, err := json.Marshal(classInfos)
		if err != nil {
			c.log.Errorw(classLog.Msg, fmt.Sprintf("json Marshal (%v) err", classInfos),
				classLog.Reason, err)
			return err
		}
		val = string(valByte)
		expire = Expiration
	}

	err := c.rdb.Set(key, val, expire).Err()
	if err != nil {
		c.log.Errorw(classLog.Msg, fmt.Sprintf("Redis:Set k(%s)-v(%s)", key, val),
			classLog.Reason, err)
		return err
	}
	return nil
}
func (c ClassInfoCacheRepo) GetClassInfosFromCache(ctx context.Context, key string) ([]*model.ClassInfo, error) {
	var classInfos = make([]*model.ClassInfo, 0)
	val, err := c.rdb.Get(key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, fmt.Errorf("error getting class info from cache: %w", err)
		}
		c.log.Errorw(classLog.Msg, fmt.Sprintf("Redis:get key(%s) err", key),
			classLog.Reason, err)
		return nil, err
	}
	if val == RedisNull {
		return nil, nil
	}
	err = json.Unmarshal([]byte(val), &classInfos)
	if err != nil {
		c.log.Errorw(classLog.Msg, fmt.Sprintf("json Unmarshal (%v) err", val),
			classLog.Reason, err)
		return nil, err
	}
	return classInfos, nil
}

func (c ClassInfoCacheRepo) DeleteClassInfoFromCache(ctx context.Context, classInfosKey ...string) error {
	if err := c.rdb.Del(classInfosKey...).Err(); err != nil {
		c.log.Errorw(classLog.Msg, fmt.Sprintf("redis delete key{%v} err", classInfosKey))
	}
	return nil
}
