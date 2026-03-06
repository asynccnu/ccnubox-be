package data

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/be-classlist/internal/conf"
	"github.com/redis/go-redis/v9"
)

type StudentAndCourseCacheRepo struct {
	rdb            *redis.Client
	metaDataExpire time.Duration
}

func NewStudentAndCourseCacheRepo(rdb *redis.Client, cf *conf.Server) *StudentAndCourseCacheRepo {
	metaDataExpire := 24 * time.Hour
	if cf.ClassExpiration > 0 {
		metaDataExpire = time.Duration(cf.ClassExpiration) * time.Second
	}
	return &StudentAndCourseCacheRepo{
		rdb:            rdb,
		metaDataExpire: metaDataExpire,
	}
}

func (s StudentAndCourseCacheRepo) generateClassMetaDataKey(stuId, xnm, xqm string) string {
	return fmt.Sprintf("ClassMetaData:%s:%s:%s", stuId, xnm, xqm)
}

// GetClassMetaData 查询单个课程元数据缓存
func (s StudentAndCourseCacheRepo) GetClassMetaData(ctx context.Context, stuId, claId, xnm, xqm string) (*ClassMetaData, error) {
	key := s.generateClassMetaDataKey(stuId, xnm, xqm)
	val, err := s.rdb.HGet(ctx, key, claId).Result()
	if err != nil {
		return nil, err
	}

	var metaData ClassMetaData
	if err := json.Unmarshal([]byte(val), &metaData); err != nil {
		return nil, err
	}

	return &metaData, nil
}

// GetSelectClassMetaData 查询学生某学期指定课程的元数据缓存
func (s StudentAndCourseCacheRepo) GetSelectClassMetaData(ctx context.Context, stuId, xnm, xqm string, claIds []string) (map[string]*ClassMetaData, error) {
	key := s.generateClassMetaDataKey(stuId, xnm, xqm)

	if len(claIds) == 0 {
		return map[string]*ClassMetaData{}, nil
	}

	// 使用HMGet批量获取指定的claIds
	vals, err := s.rdb.HMGet(ctx, key, claIds...).Result()
	if err != nil {
		return nil, err
	}

	metaDataMap := make(map[string]*ClassMetaData)
	for i, val := range vals {
		if val == nil {
			continue
		}
		valStr, ok := val.(string)
		if !ok {
			continue
		}
		var metaData ClassMetaData
		if err := json.Unmarshal([]byte(valStr), &metaData); err != nil {
			continue
		}
		metaDataMap[claIds[i]] = &metaData
	}

	return metaDataMap, nil
}

// SetClassMetaData 设置单个课程元数据缓存
func (s StudentAndCourseCacheRepo) SetClassMetaData(ctx context.Context, stuId, claId, xnm, xqm string, metaData *ClassMetaData) error {
	key := s.generateClassMetaDataKey(stuId, xnm, xqm)
	data, err := json.Marshal(metaData)
	if err != nil {
		return err
	}

	if err := s.rdb.HSet(ctx, key, claId, data).Err(); err != nil {
		return err
	}

	// 设置过期时间
	return s.rdb.Expire(ctx, key, s.metaDataExpire).Err()
}

// SetAllClassMetaData 批量设置课程元数据缓存
func (s StudentAndCourseCacheRepo) SetAllClassMetaData(ctx context.Context, stuId, xnm, xqm string, metaDataMap map[string]*ClassMetaData) error {
	key := s.generateClassMetaDataKey(stuId, xnm, xqm)

	// 构建哈希字段
	fields := make(map[string]interface{}, len(metaDataMap))
	for claId, metaData := range metaDataMap {
		data, err := json.Marshal(metaData)
		if err != nil {
			return err
		}
		fields[claId] = data
	}

	if len(fields) == 0 {
		return nil
	}

	if err := s.rdb.HSet(ctx, key, fields).Err(); err != nil {
		return err
	}

	// 设置过期时间
	return s.rdb.Expire(ctx, key, s.metaDataExpire).Err()
}

// DeleteClassMetaData 删除单个课程元数据缓存
func (s StudentAndCourseCacheRepo) DeleteClassMetaData(ctx context.Context, stuId, claId, xnm, xqm string) error {
	key := s.generateClassMetaDataKey(stuId, xnm, xqm)
	return s.rdb.HDel(ctx, key, claId).Err()
}

// DeleteAllClassMetaData 删除学生某学期所有课程元数据缓存
func (s StudentAndCourseCacheRepo) DeleteAllClassMetaData(ctx context.Context, stuId, xnm, xqm string) error {
	key := s.generateClassMetaDataKey(stuId, xnm, xqm)
	return s.rdb.Del(ctx, key).Err()
}
