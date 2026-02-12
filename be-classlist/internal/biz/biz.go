package biz

import (
	"context"
	"time"

	"github.com/asynccnu/ccnubox-be/be-classlist/internal/data/do"
	"github.com/google/wire"
)

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(NewClassUsecase)

type ClassCrawler interface {
	// 获取本科生的课表
	GetClassInfosForUndergraduate(ctx context.Context, stuID, year, semester, cookie string) ([]*ClassInfo, []*StudentCourse, int, error)
	// 获取研究生的课表(未实现)
	GetClassInfoForGraduateStudent(ctx context.Context, stuID, year, semester, cookie string) ([]*ClassInfo, []*StudentCourse, int, error)
}

type ClassRepo interface {
	GetClassesFromLocal(ctx context.Context, stuID, year, semester string) ([]*ClassInfo, error)
	GetSpecificClassInfo(ctx context.Context, classID string) (*ClassInfo, error)
	AddClass(ctx context.Context, stuID, year, semester string, classInfo *ClassInfo, sc *StudentCourse) error
	DeleteClass(ctx context.Context, stuID, year, semester string, classID []string) error
	GetRecycledIds(ctx context.Context, stuID, year, semester string) ([]string, error)
	IsRecycledCourseManual(ctx context.Context, stuID, year, semester, classID string) bool
	CheckClassIdIsInRecycledBin(ctx context.Context, stuID, year, semester, classID string) bool
	RemoveClassFromRecycledBin(ctx context.Context, stuID, year, semester, classID string) error
	UpdateClass(ctx context.Context, stuID, year, semester, oldClassID string,
		newClassInfo *ClassInfo, newSc *StudentCourse) error
	SaveClass(ctx context.Context, stuID, year, semester string, classInfos []*ClassInfo, scs []*StudentCourse) error
	CheckSCIdsExist(ctx context.Context, stuID, year, semester, classID string) bool
	GetAllSchoolClassInfos(ctx context.Context, year, semester string, cursor time.Time) []*ClassInfo
	GetAddedClasses(ctx context.Context, stuID, year, semester string) ([]*ClassInfo, error)
	IsClassOfficial(ctx context.Context, stuID, year, semester, classID string) bool
	GetClassNote(ctx context.Context, stuID, year, semester, classID string) string
	UpdateClassNote(ctx context.Context, stuID, year, semester, classID, note string) error
	GetClassNatures(ctx context.Context, stuID string) []string
}

type JxbRepo interface {
	// 保存教学班
	SaveJxb(ctx context.Context, stuID string, jxbID []string) error
	// 根据教学班ID查询stuID
	FindStuIdsByJxbId(ctx context.Context, jxbId string) ([]string, error)
}
type CCNUServiceProxy interface {
	// 从其他服务获取cookie
	GetCookie(ctx context.Context, stuID string) (string, error)
}

type RefreshLogRepo interface {
	InsertRefreshLog(ctx context.Context, stuID, year, semester string) (uint64, error)
	UpdateRefreshLogStatus(ctx context.Context, logID uint64, status string) error
	SearchRefreshLog(ctx context.Context, stuID, year, semester string) (*do.ClassRefreshLog, error)
	GetRefreshLogByID(ctx context.Context, logID uint64) (*do.ClassRefreshLog, error)
	GetLastRefreshTime(ctx context.Context, stuID, year, semester string, beforeTime time.Time) *time.Time
	DeleteRedundantLogs(ctx context.Context, stuID, year, semester string) error
}

type DelayQueue interface {
	Send(ctx context.Context, key, value []byte) error
	Consume(groupID string, f func(ctx context.Context, key []byte, value []byte)) error
	Close()
}
