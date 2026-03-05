package data

import (
	"fmt"

	"github.com/asynccnu/ccnubox-be/be-classlist/internal/conf"
	"github.com/redis/go-redis/v9"
)

type StudentAndCourseCacheRepo struct {
	rdb *redis.Client
}

func NewStudentAndCourseCacheRepo(rdb *redis.Client, cf *conf.Server) *StudentAndCourseCacheRepo {
	return &StudentAndCourseCacheRepo{
		rdb: rdb,
	}
}

func (s StudentAndCourseCacheRepo) GenerateClassInfosKey(stuId, xnm, xqm string) string {
	return fmt.Sprintf("ClassInfos:%s:%s:%s", stuId, xnm, xqm)
}
