package data

import (
	"context"
	"fmt"
	"github.com/asynccnu/ccnubox-be/Muxi_ClassList/internal/classLog"
	"github.com/go-redis/redis"
)

type StudentAndCourseCacheRepo struct {
	rdb *redis.Client
	log classLog.Clogger
}

func NewStudentAndCourseCacheRepo(rdb *redis.Client, logger classLog.Clogger) *StudentAndCourseCacheRepo {
	return &StudentAndCourseCacheRepo{
		rdb: rdb,
		log: logger,
	}
}

func (s StudentAndCourseCacheRepo) GetRecycledClassIds(ctx context.Context, key string) ([]string, error) {
	res, err := s.rdb.SMembers(key).Result()
	if err != nil {
		s.log.Errorw(classLog.Msg, fmt.Sprintf("Redis:get classIds From set(%s)", key),
			classLog.Reason, err)
		return nil, err
	}
	return res, nil
}
func (s StudentAndCourseCacheRepo) CheckRecycleIdIsExist(ctx context.Context, RecycledBinKey, classId string) bool {
	exists, err := s.rdb.SIsMember(RecycledBinKey, classId).Result()
	if err != nil {
		s.log.Errorw(classLog.Msg, fmt.Sprintf("Redis:check classId(%s) is exist in RecycleBinKey(%s) err", classId, RecycledBinKey),
			classLog.Reason, err)
		return false
	}
	return exists
}
func (s StudentAndCourseCacheRepo) RemoveClassFromRecycledBin(ctx context.Context, RecycledBinKey, classId string) error {
	_, err := s.rdb.SRem(RecycledBinKey, classId).Result()
	if err != nil {
		s.log.Errorw(classLog.Msg, fmt.Sprintf("Redis:remove member(%s) from set(%s) err", classId, RecycledBinKey),
			classLog.Reason, err)
		return err
	}
	return nil
}

func (s StudentAndCourseCacheRepo) RecycleClassId(ctx context.Context, recycleBinKey string, classId ...string) error {

	// 将 ClassId 放入回收站
	if err := s.rdb.SAdd(recycleBinKey, classId).Err(); err != nil {
		s.log.Errorw(classLog.Msg, fmt.Sprintf("Redis:Add classId(%s) to set(%s) err", classId, recycleBinKey),
			classLog.Reason, err)
		return err
	}

	// 设置回收站的过期时间
	if err := s.rdb.Expire(recycleBinKey, RecycleExpiration).Err(); err != nil {
		s.log.Errorw(classLog.Msg, "Redis:set expire err",
			classLog.Reason, err)
		return err
	}
	return nil
}
