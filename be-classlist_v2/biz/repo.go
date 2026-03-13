package biz

import (
	"context"
	"time"

	"github.com/asynccnu/ccnubox-be/be-classlist_v2/biz/model"
)

type ClassRepo interface {
	GetClassesFromLocal(ctx context.Context, stuID, year, semester string) ([]*model.ClassInfoBO, error)
}

type RefreshLogRepo interface {
	InsertRefreshLog(ctx context.Context, stuID, year, semester, status string, logTime time.Time) (uint64, error)
	UpdateRefreshLogStatus(ctx context.Context, logID uint64, status string) error
	SearchNewestRefreshLog(ctx context.Context, stuID, year, semester string, endTime time.Time) (*model.ClassRefreshLogBO, error)
	GetRefreshLogByID(ctx context.Context, logID uint64) (*model.ClassRefreshLogBO, error)
	GetLastRefreshTime(ctx context.Context, stuID, year, semester, status string, beforeTime time.Time) *time.Time
}
