package repo

import (
	"context"
	"time"

	"github.com/asynccnu/ccnubox-be/be-classlist_v2/repository/dao"
)

type RefreshLogRepo struct {
	refDB *dao.RefreshLogDAO
}

func (repo *RefreshLogRepo) GetLastRefreshTime(ctx context.Context, stuID, year, semester, status string, beforeTime time.Time) (*time.Time, error) {
	return repo.GetLastRefreshTime(ctx, stuID, year, semester, status, beforeTime)
}
