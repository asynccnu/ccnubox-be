package biz

import (
	"context"
	"time"

	"github.com/asynccnu/ccnubox-be/be-classlist_v2/biz/model"
)

type ClassRepo interface {
	GetClassesFromLocal(ctx context.Context, stuID, year, semester string) ([]*model.ClassInfoBO, error)
	CacheClass(ctx context.Context, stuID, year, semester string)
	GetSpecificClassInfo(ctx context.Context, stuID, year, semester, classID string) (*model.ClassInfoBO, error)
	AddClass(ctx context.Context, stuID, year, semester string, classInfo *model.ClassInfoBO, sc *model.StudentCourseBO) error
	DeleteClass(ctx context.Context, stuID, year, semester string, classInfo *model.ClassInfoBO) error

	UpdateClass(ctx context.Context, stuID, year, semester, oldClassID string,
		newClassInfo *model.ClassInfoBO, newSc *model.StudentCourseBO) error
	SaveClass(ctx context.Context, stuID, year, semester string, classInfos []*model.ClassInfoBO, scs []*model.StudentCourseBO) error
	CheckSCIdsExist(ctx context.Context, stuID, year, semester, classID string) bool
	GetAllSchoolClassInfos(ctx context.Context, year, semester string, cursor time.Time) []*model.ClassInfoBO
	GetAddedClasses(ctx context.Context, stuID, year, semester string) ([]*model.ClassInfoBO, error)
	GetClassMetaData(ctx context.Context, stuID, year, semester, classID string) (*model.ClassMetaDataBO, error)
	UpdateClassNote(ctx context.Context, stuID, year, semester, classID, note string) error
	GetClassNatures(ctx context.Context, stuID string) []string
	GetStudentIDs(ctx context.Context, lastStuID string, size int) ([]string, error)
}

type RefreshLogRepo interface {
	InsertRefreshLog(ctx context.Context, stuID, year, semester, status string, logTime time.Time) (uint64, error)
	UpdateRefreshLogStatus(ctx context.Context, logID uint64, status string) error
	SearchNewestRefreshLog(ctx context.Context, stuID, year, semester string, endTime time.Time) (*model.ClassRefreshLogBO, error)
	GetRefreshLogByID(ctx context.Context, logID uint64) (*model.ClassRefreshLogBO, error)
	GetLastRefreshTime(ctx context.Context, stuID, year, semester, status string, beforeTime time.Time) (*time.Time, error)
}
type CCNUService interface {
	GetCookie(ctx context.Context, stuID string) (string, error)
}

type ClassCrawler interface {
	GetClassInfosForUndergraduate(ctx context.Context, stuID, year, semester, cookie string) ([]*model.ClassInfoBO, []*model.StudentCourseBO, int, error)
	GetClassInfoForGraduateStudent(ctx context.Context, stuID, year, semester, cookie string) ([]*model.ClassInfoBO, []*model.StudentCourseBO, int, error)
}

type DelayQueue interface {
	Send(ctx context.Context, key, value []byte) error
	Consume(groupID string, f func(ctx context.Context, key []byte, value []byte)) error
	Close()
}
type JxbRepo interface {
	SaveJxb(ctx context.Context, stuID string, jxbID []string) error
	FindStuIdsByJxbId(ctx context.Context, jxbId string) ([]string, error)
}
