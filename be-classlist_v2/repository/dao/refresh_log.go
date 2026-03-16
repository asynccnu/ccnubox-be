package dao

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/be-classlist_v2/repository/model"

	"gorm.io/gorm"
)

type RefreshLogDAO struct {
	db *gorm.DB
}

func NewRefreshLogRepo(db *gorm.DB) *RefreshLogDAO {
	return &RefreshLogDAO{
		db: db,
	}
}

// GetLastRefreshTime 返回最后一次刷新成功的时间
func (r *RefreshLogDAO) GetLastRefreshTime(ctx context.Context, stuID, year, semester, status string, beforeTime time.Time) (*time.Time, error) {
	var refreshLog model.ClassRefreshLog
	err := r.db.WithContext(ctx).Table(model.ClassRefreshLogTableName).
		Where("stu_id = ? and year = ? and semester = ? and updated_at < ? and status = ?", stuID, year, semester, beforeTime, status).
		Order("updated_at desc").First(&refreshLog).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get last refresh time: %w", err)
	}
	return &refreshLog.UpdatedAt, nil
}
