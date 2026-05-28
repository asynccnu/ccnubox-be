package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/be-classlist_v2/conf"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/repository/model"
)

type StudentCourseCache struct {
	BaseCache
	metaDataExpire time.Duration
}

func NewStudentCourseCache(base BaseCache, cf *conf.ServerConf) *StudentCourseCache {
	metaDataExpire := 24 * time.Hour
	if cf.ClassListConf.ClassExpiration > 0 {
		metaDataExpire = time.Duration(cf.ClassListConf.ClassExpiration) * time.Millisecond
	}
	return &StudentCourseCache{
		BaseCache:      base,
		metaDataExpire: metaDataExpire,
	}
}

func (s StudentCourseCache) generateClassMetaDataKey(stuId, xnm, xqm string) string {
	return fmt.Sprintf("ClassMetaData:%s:%s:%s", stuId, xnm, xqm)
}

// GetSelectClassMetaData 查询学生某学期指定课程的元数据缓存
func (s StudentCourseCache) GetSelectClassMetaData(ctx context.Context, stuId, xnm, xqm string, claIds []string) (map[string]*model.ClassMetaData, error) {
	key := s.generateClassMetaDataKey(stuId, xnm, xqm)

	if len(claIds) == 0 {
		return map[string]*model.ClassMetaData{}, nil
	}

	// 使用HMGet批量获取指定的claIds
	vals, err := s.rdb.HMGet(ctx, key, claIds...).Result()
	if err != nil {
		return nil, err
	}

	metaDataMap := make(map[string]*model.ClassMetaData)
	for i, val := range vals {
		if val == nil {
			continue
		}
		valStr, ok := val.(string)
		if !ok {
			continue
		}
		var metaData model.ClassMetaData
		if err := json.Unmarshal([]byte(valStr), &metaData); err != nil {
			continue
		}
		metaDataMap[claIds[i]] = &metaData
	}

	return metaDataMap, nil
}

// SetAllClassMetaData 批量设置课程元数据缓存
func (s StudentCourseCache) SetAllClassMetaData(ctx context.Context, stuId, xnm, xqm string, metaDataMap map[string]*model.ClassMetaData) error {
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

// DeleteAllClassMetaData 删除学生某学期所有课程元数据缓存
func (s StudentCourseCache) DeleteAllClassMetaData(ctx context.Context, stuId, xnm, xqm string) error {
	key := s.generateClassMetaDataKey(stuId, xnm, xqm)
	return s.rdb.Del(ctx, key).Err()
}
