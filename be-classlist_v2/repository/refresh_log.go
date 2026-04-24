package repo

import (
	"context"
	"time"

	"github.com/asynccnu/ccnubox-be/be-classlist_v2/biz"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/biz/model"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/repository/dao"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

type RefreshLogRepo struct {
	refDB *dao.RefreshLogDAO
	log   logger.Logger
}

func NewRefreshLogRepo(refDB *dao.RefreshLogDAO, l logger.Logger) biz.RefreshLogRepo {
	return &RefreshLogRepo{
		refDB: refDB,
		log:   l,
	}
}

func (r *RefreshLogRepo) GetLastRefreshTime(ctx context.Context, stuID, year, semester, status string, beforeTime time.Time) (*time.Time, error) {
	return r.refDB.GetLastRefreshTime(ctx, stuID, year, semester, status, beforeTime)
}

func (r *RefreshLogRepo) InsertRefreshLog(ctx context.Context, stuID, year, semester, status string, logTime time.Time) (uint64, error) {
	return r.refDB.InsertRefreshLog(ctx, stuID, year, semester, status, logTime)
}

func (r *RefreshLogRepo) UpdateRefreshLogStatus(ctx context.Context, logID uint64, status string) error {
	return r.refDB.UpdateRefreshLogStatus(ctx, logID, status)
}

func (r *RefreshLogRepo) SearchNewestRefreshLog(ctx context.Context, stuID, year, semester string, endTime time.Time) (*model.ClassRefreshLogBO, error) {
	DO, err := r.refDB.SearchNewestRefreshLog(ctx, stuID, year, semester, endTime)
	if err != nil {
		return nil, err
	}
	return ClassRefreshLogDOToBO(DO), nil
}

func (r *RefreshLogRepo) GetRefreshLogByID(ctx context.Context, logID uint64) (*model.ClassRefreshLogBO, error) {
	DO, err := r.refDB.GetRefreshLogByID(ctx, logID)
	if err != nil {
		return nil, err
	}
	return ClassRefreshLogDOToBO(DO), nil
}
