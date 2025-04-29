package biz

import (
	"context"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/model"
)

type StudentAndCourseDBRepo interface {
	SaveManyStudentAndCourseToDB(ctx context.Context, scs []*model.StudentCourse) error
	SaveStudentAndCourseToDB(ctx context.Context, sc *model.StudentCourse) error
	DeleteStudentAndCourseInDB(ctx context.Context, stuID, year, semester string, claID []string) error
	DeleteStudentAndCourseByTimeFromDB(ctx context.Context, stuID, year, semester string) error
	CheckExists(ctx context.Context, xnm, xqm, stuId, classId string) bool
	GetClassNum(ctx context.Context, stuID, year, semester string, isManuallyAdded bool) (num int64, err error)
	CheckManualCourseStatus(ctx context.Context, stuID, year, semester, classID string) bool
}

type StudentAndCourseCacheRepo interface {
	GetRecycledClassIds(ctx context.Context, key string) ([]string, error)
	RecycleClassId(ctx context.Context, recycleBinKey string, classId string, isAdded bool) error
	CheckRecycleIdIsExist(ctx context.Context, RecycledBinKey, classId string) bool
	RemoveClassFromRecycledBin(ctx context.Context, RecycledBinKey, classId string) error
	IsRecycledCourseManual(ctx context.Context, RecycledBinKey, classId string) bool
}

type StudentAndCourseRepo struct {
	DB    StudentAndCourseDBRepo
	Cache StudentAndCourseCacheRepo
}

func NewStudentAndCourseRepo(DB StudentAndCourseDBRepo, Cache StudentAndCourseCacheRepo) *StudentAndCourseRepo {
	return &StudentAndCourseRepo{
		DB:    DB,
		Cache: Cache,
	}
}
