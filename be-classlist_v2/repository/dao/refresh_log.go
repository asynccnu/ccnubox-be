package dao

import (
	"context"
	"errors"
	"time"

	"github.com/asynccnu/ccnubox-be/be-classlist_v2/repository/model"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"gorm.io/gorm"
)

type RefreshLogDAO struct {
	BaseDAO
}

func NewRefreshLogDAO(base BaseDAO) *RefreshLogDAO {
	return &RefreshLogDAO{
		BaseDAO: base,
	}
}

// GetLastRefreshTime 返回最后一次刷新成功的时间
func (r *RefreshLogDAO) GetLastRefreshTime(ctx context.Context, stuID, year, semester, status string, beforeTime time.Time) (*time.Time, error) {
	var refreshLog model.ClassRefreshLog
	err := r.GetDB(ctx).WithContext(ctx).Table(model.ClassRefreshLogTableName).
		Where("stu_id = ? and year = ? and semester = ? and updated_at < ? and status = ?", stuID, year, semester, beforeTime, status).
		Order("updated_at desc").First(&refreshLog).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, errorx.Errorf("dao.refreshLog.GetLastRefreshTime: stuID=%s, year=%s, semester=%s, status=%s: %w", stuID, year, semester, status, err)
	}
	return &refreshLog.UpdatedAt, nil
}

// InsertRefreshLog 插入一条刷新记录
func (r *RefreshLogDAO) InsertRefreshLog(ctx context.Context, stuID, year, semester, status string, logTime time.Time) (uint64, error) {
	refreshLog := model.ClassRefreshLog{
		StuID:     stuID,
		Year:      year,
		Semester:  semester,
		Status:    status,
		UpdatedAt: logTime,
	}
	err := r.createRefreshLog(ctx, &refreshLog)
	if err != nil {
		return 0, errorx.Errorf("dao.refreshLog.InsertRefreshLog: stuID=%s, year=%s, semester=%s, status=%s: %w", stuID, year, semester, status, err)
	}
	return refreshLog.ID, nil
}

func (r *RefreshLogDAO) UpdateRefreshLogStatus(ctx context.Context, logID uint64, status string) error {
	err := r.GetDB(ctx).WithContext(ctx).Table(model.ClassRefreshLogTableName).
		Where("id = ?", logID).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
	if err != nil {
		return errorx.Errorf("dao.refreshLog.UpdateRefreshLogStatus: logID=%d, status=%s: %w", logID, status, err)
	}
	return nil
}

// SearchNewestRefreshLog 查找在指定时间内的最新的一条记录
func (r *RefreshLogDAO) SearchNewestRefreshLog(ctx context.Context, stuID, year, semester string, endTime time.Time) (*model.ClassRefreshLog, error) {
	var refreshLog model.ClassRefreshLog
	err := r.GetDB(ctx).WithContext(ctx).Table(model.ClassRefreshLogTableName).
		Where("stu_id = ? and year = ? and semester = ? and updated_at < ?", stuID, year, semester, endTime).
		Order("updated_at desc").First(&refreshLog).Error
	if err != nil {
		return nil, errorx.Errorf("dao.refreshLog.SearchNewestRefreshLog: stuID=%s, year=%s, semester=%s: %w", stuID, year, semester, err)
	}
	return &refreshLog, nil
}

// GetRefreshLogByID  查找指定ID的记录
func (r *RefreshLogDAO) GetRefreshLogByID(ctx context.Context, logID uint64) (*model.ClassRefreshLog, error) {
	var refreshLog model.ClassRefreshLog
	err := r.GetDB(ctx).WithContext(ctx).Table(model.ClassRefreshLogTableName).
		Where("id = ?", logID).First(&refreshLog).Error
	if err != nil {
		return nil, errorx.Errorf("dao.refreshLog.GetRefreshLogByID: logID=%d: %w", logID, err)
	}
	return &refreshLog, nil
}

func (r *RefreshLogDAO) createRefreshLog(ctx context.Context, refreshLog *model.ClassRefreshLog) error {
	if err := r.GetDB(ctx).WithContext(ctx).Create(refreshLog).Error; err != nil {
		return errorx.Errorf("dao.refreshLog.createRefreshLog: %w", err)
	}
	return nil
}
