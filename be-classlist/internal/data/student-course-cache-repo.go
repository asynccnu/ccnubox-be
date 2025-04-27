package data

import (
	"context"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	"time"
)

type StudentAndCourseCacheRepo struct {
	rdb               *redis.Client
	recycleExpiration time.Duration
	log               *log.Helper
}

func NewStudentAndCourseCacheRepo(rdb *redis.Client, cf *conf.Server, logger log.Logger) *StudentAndCourseCacheRepo {
	expire := 30 * 24 * time.Hour

	if cf.RecycleExpiration > 0 {
		expire = time.Duration(cf.RecycleExpiration) * time.Second
	}

	return &StudentAndCourseCacheRepo{
		rdb:               rdb,
		log:               log.NewHelper(logger),
		recycleExpiration: expire,
	}
}

func (s StudentAndCourseCacheRepo) GetRecycledClassIds(ctx context.Context, key string) ([]string, error) {
	res, err := s.rdb.SMembers(ctx, key).Result()
	if err != nil {
		s.log.Errorf("redis: getrecycledClassIds key = %v failed: %v", key, err)

		return nil, err
	}
	return res, nil
}
func (s StudentAndCourseCacheRepo) CheckRecycleIdIsExist(ctx context.Context, RecycledBinKey, classId string) bool {
	exists, err := s.rdb.SIsMember(ctx, RecycledBinKey, classId).Result()
	if err != nil {
		s.log.Errorf("redis: check classId(%s) in set(%s) failed: %v", classId, RecycledBinKey, err)
		return false
	}
	return exists
}
func (s StudentAndCourseCacheRepo) RemoveClassFromRecycledBin(ctx context.Context, RecycledBinKey, classId string) error {
	_, err := s.rdb.SRem(ctx, RecycledBinKey, classId).Result()
	if err != nil {
		s.log.Errorf("redis: remove classId(%s) from set(%s) failed: %v", classId, RecycledBinKey, err)
		return err
	}
	return nil
}

func (s StudentAndCourseCacheRepo) RecycleClassId(ctx context.Context, recycleBinKey string, classId ...string) error {

	// 将 ClassId 放入回收站
	if err := s.rdb.SAdd(ctx, recycleBinKey, classId).Err(); err != nil {
		s.log.Errorf("redis: add classId(%s) to set(%s) failed: %v", classId, recycleBinKey, err)
		return err
	}

	// 设置回收站的过期时间
	if err := s.rdb.Expire(ctx, recycleBinKey, s.recycleExpiration).Err(); err != nil {
		s.log.Errorf("redis: set expiration for key(%s) failed: %v", recycleBinKey, err)
		return err
	}
	return nil
}
