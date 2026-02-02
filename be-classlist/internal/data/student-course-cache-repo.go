package data

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/be-classlist/internal/conf"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/redis/go-redis/v9"
)

type StudentAndCourseCacheRepo struct {
	rdb               *redis.Client
	recycleExpiration time.Duration
}

type RecycleClassInfo struct {
	ClassId string `json:"classId"`
	IsAdded bool   `json:"isAdded"`
}

func NewStudentAndCourseCacheRepo(rdb *redis.Client, cf *conf.Server) *StudentAndCourseCacheRepo {
	expire := 30 * 24 * time.Hour

	if cf.RecycleExpiration > 0 {
		expire = time.Duration(cf.RecycleExpiration) * time.Second
	}

	return &StudentAndCourseCacheRepo{
		rdb:               rdb,
		recycleExpiration: expire,
	}
}

func (s StudentAndCourseCacheRepo) GetRecycledClassIds(ctx context.Context, key string) ([]string, error) {
	logh := logger.GetLoggerFromCtx(ctx)
	members, err := s.rdb.SMembers(ctx, key).Result()
	if err != nil {
		logh.Errorf("redis: getrecycledClassIds key = %v failed: %v", key, err)
		return nil, err
	}
	ids := make([]string, 0, len(members))
	for _, member := range members {
		var recycledClass RecycleClassInfo
		err = json.Unmarshal([]byte(member), &recycledClass)
		if err != nil {
			logh.Errorf("redis: getrecycledClassIds key = %v failed: %v", key, err)
			return nil, err
		}
		ids = append(ids, recycledClass.ClassId)
	}
	return ids, nil
}

func (s StudentAndCourseCacheRepo) CheckRecycleIdIsExist(ctx context.Context, RecycledBinKey, classId string) bool {
	logh := logger.GetLoggerFromCtx(ctx)
	members, err := s.rdb.SMembers(ctx, RecycledBinKey).Result()
	if err != nil {
		logh.Errorf("redis: get members of set(%s) failed: %v", RecycledBinKey, err)
		return false
	}

	for _, member := range members {
		var recycledClass RecycleClassInfo
		err = json.Unmarshal([]byte(member), &recycledClass)
		if err != nil {
			logh.Errorf("redis: get member(%s) failed: %v", member, err)
			continue
		}
		if recycledClass.ClassId == classId {
			return true
		}
	}
	return false
}

func (s StudentAndCourseCacheRepo) IsRecycledCourseManual(ctx context.Context, RecycledBinKey, classId string) bool {
	logh := logger.GetLoggerFromCtx(ctx)
	members, err := s.rdb.SMembers(ctx, RecycledBinKey).Result()
	if err != nil {
		logh.Errorf("redis: get members of set(%s) failed: %v", RecycledBinKey, err)
		return false
	}
	for _, member := range members {
		var recycledClass RecycleClassInfo
		err = json.Unmarshal([]byte(member), &recycledClass)
		if err != nil {
			logh.Errorf("redis: get member(%s) failed: %v", member, err)
			continue
		}
		if recycledClass.ClassId == classId {
			return recycledClass.IsAdded
		}
	}
	return false
}

func (s StudentAndCourseCacheRepo) RemoveClassFromRecycledBin(ctx context.Context, RecycledBinKey, classId string) error {
	logh := logger.GetLoggerFromCtx(ctx)
	members, err := s.rdb.SMembers(ctx, RecycledBinKey).Result()
	if err != nil {
		logh.Errorf("redis: get members of set(%s) failed: %v", RecycledBinKey, err)
		return err
	}

	for _, member := range members {
		var recycleInfo RecycleClassInfo
		if err := json.Unmarshal([]byte(member), &recycleInfo); err != nil {
			logh.Errorf("redis: unmarshal recycleInfo(%s) failed: %v", member, err)
			continue
		}
		if recycleInfo.ClassId == classId {
			if err := s.rdb.SRem(ctx, RecycledBinKey, member).Err(); err != nil {
				logh.Errorf("redis: remove recycleInfo(%s) failed: %v", member, err)
				return err
			}
			logh.Infof("redis: classId(%s) removed from set(%s)", classId, RecycledBinKey)
			break
		}
	}
	return nil
}

func (s StudentAndCourseCacheRepo) RecycleClassId(ctx context.Context, recycleBinKey string, classId string, isAdded bool) error {
	logh := logger.GetLoggerFromCtx(ctx)
	val := RecycleClassInfo{ClassId: classId, IsAdded: isAdded}

	jsonVal, err := json.Marshal(val)
	if err != nil {
		return err
	}

	// 将 ClassId 放入回收站
	if err := s.rdb.SAdd(ctx, recycleBinKey, jsonVal).Err(); err != nil {
		logh.Errorf("redis: add classId(%s) to set(%s) failed: %v", classId, recycleBinKey, err)
		return err
	}

	// 设置回收站的过期时间
	if err := s.rdb.Expire(ctx, recycleBinKey, s.recycleExpiration).Err(); err != nil {
		logh.Errorf("redis: set expiration for key(%s) failed: %v", recycleBinKey, err)
		return err
	}
	return nil
}

func (s StudentAndCourseCacheRepo) GenerateRecycleSetName(stuId, xnm, xqm string) string {
	return fmt.Sprintf("Recycle:%s:%s:%s", stuId, xnm, xqm)
}

func (s StudentAndCourseCacheRepo) GenerateClassInfosKey(stuId, xnm, xqm string) string {
	return fmt.Sprintf("ClassInfos:%s:%s:%s", stuId, xnm, xqm)
}
