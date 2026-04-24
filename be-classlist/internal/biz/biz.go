package biz

import (
	"context"
	"time"

	"github.com/google/wire"
)

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(NewClassUsecase)

type ClassCrawler interface {
	GetClassInfosForUndergraduate(ctx context.Context, stuID, year, semester, cookie string) ([]*ClassInfoBO, []*StudentCourse, int, error)
	GetClassInfoForGraduateStudent(ctx context.Context, stuID, year, semester, cookie string) ([]*ClassInfoBO, []*StudentCourse, int, error)
}

type ClassRepo interface {
	GetClassesFromLocal(ctx context.Context, stuID, year, semester string) ([]*ClassInfoBO, error)
	CacheClass(ctx context.Context, stuID, year, semester string)
	GetSpecificClassInfo(ctx context.Context, stuID, year, semester, classID string) (*ClassInfoBO, error)
	AddClass(ctx context.Context, stuID, year, semester string, classInfo *ClassInfoBO, sc *StudentCourse) error
	DeleteClass(ctx context.Context, stuID, year, semester string, classInfo *ClassInfoBO) error

	UpdateClass(ctx context.Context, stuID, year, semester, oldClassID string,
		newClassInfo *ClassInfoBO, newSc *StudentCourse) error
	SaveClass(ctx context.Context, stuID, year, semester string, classInfos []*ClassInfoBO, scs []*StudentCourse) error
	CheckSCIdsExist(ctx context.Context, stuID, year, semester, classID string) bool
	GetAllSchoolClassInfos(ctx context.Context, year, semester string, cursor time.Time) []*ClassInfoBO
	GetAddedClasses(ctx context.Context, stuID, year, semester string) ([]*ClassInfoBO, error)
	GetClassMetaData(ctx context.Context, stuID, year, semester, classID string) (ClassMetaDataBO, error)
	UpdateClassNote(ctx context.Context, stuID, year, semester, classID, note string) error
	GetClassNatures(ctx context.Context, stuID string) []string
	GetStudentIDs(ctx context.Context, lastStuID string, size int) ([]string, error)
}

type JxbRepo interface {
	SaveJxb(ctx context.Context, stuID string, jxbID []string) error
	FindStuIdsByJxbId(ctx context.Context, jxbId string) ([]string, error)
}

type CCNUServiceProxy interface {
	GetCookie(ctx context.Context, stuID string) (string, error)
}

type RefreshLogRepo interface {
	InsertRefreshLog(ctx context.Context, stuID, year, semester, status string, logTime time.Time) (uint64, error)
	UpdateRefreshLogStatus(ctx context.Context, logID uint64, status string) error
	SearchNewestRefreshLog(ctx context.Context, stuID, year, semester string, endTime time.Time) (*ClassRefreshLogBO, error)
	GetRefreshLogByID(ctx context.Context, logID uint64) (*ClassRefreshLogBO, error)
	GetLastRefreshTime(ctx context.Context, stuID, year, semester, status string, beforeTime time.Time) *time.Time
}

type DelayQueue interface {
	Send(ctx context.Context, key, value []byte) error
	Consume(groupID string, f func(ctx context.Context, key []byte, value []byte)) error
	Close()
}

type RecycleBinRepo interface {
	RecycleClass(ctx context.Context, stuID, year, semester, classID string, info *ClassInfoBO) error
	RemoveClass(ctx context.Context, stuID, year, semester, classID string) error
	GetClass(ctx context.Context, stuID, year, semester, classID string) (*ClassInfoBO, bool)
	ListClasses(ctx context.Context, stuID, year, semester string) ([]*ClassInfoBO, error)
	HasClass(ctx context.Context, stuID, year, semester, classID string) bool
	CleanExpired(ctx context.Context, stuID, year, semester string) error
}
