package data

import (
	"context"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/biz"
	"time"

	"gorm.io/gorm"
)

type RefreshLogRepo struct {
	db *gorm.DB
}

func NewRefreshLogRepo(db *gorm.DB) *RefreshLogRepo {
	return &RefreshLogRepo{
		db: db,
	}
}

// InsertRefreshLog 插入一条刷新记录
func (r *RefreshLogRepo) InsertRefreshLog(ctx context.Context, stuID, year, semester, status string, logTime time.Time) (uint64, error) {
	refreshLog := ClassRefreshLog{
		StuID:     stuID,
		Year:      year,
		Semester:  semester,
		Status:    status,
		UpdatedAt: logTime,
	}
	err := r.createRefreshLog(ctx, r.db, &refreshLog)
	if err != nil {
		return 0, err
	}
	return refreshLog.ID, nil
}

func (r *RefreshLogRepo) UpdateRefreshLogStatus(ctx context.Context, logID uint64, status string) error {
	return r.db.WithContext(ctx).Table(ClassRefreshLogTableName).
		Where("id = ?", logID).Update("status", status).Error
}

// SearchNewestRefreshLog 查找在指定时间内的最新的一条记录
func (r *RefreshLogRepo) SearchNewestRefreshLog(ctx context.Context, stuID, year, semester string, endTime time.Time) (*biz.ClassRefreshLogBO, error) {
	var refreshLog ClassRefreshLog
	err := r.db.WithContext(ctx).Table(ClassRefreshLogTableName).
		Where("stu_id = ? and year = ? and semester = ? and updated_at < ?", stuID, year, semester, endTime).
		Order("updated_at desc").First(&refreshLog).Error
	if err != nil {
		return nil, err
	}
	return ClassRefreshLogDOToBO(&refreshLog), nil
}

// GetLastRefreshTime 返回最后一次刷新成功的时间
func (r *RefreshLogRepo) GetLastRefreshTime(ctx context.Context, stuID, year, semester, status string, beforeTime time.Time) *time.Time {
	var refreshLog ClassRefreshLog
	err := r.db.WithContext(ctx).Table(ClassRefreshLogTableName).
		Where("stu_id = ? and year = ? and semester = ? and updated_at < ? and status = ?", stuID, year, semester, beforeTime, status).
		Order("updated_at desc").First(&refreshLog).Error
	if err != nil {
		return nil
	}
	return &refreshLog.UpdatedAt
}

// GetRefreshLogByID  查找指定ID的记录
func (r *RefreshLogRepo) GetRefreshLogByID(ctx context.Context, logID uint64) (*biz.ClassRefreshLogBO, error) {
	var refreshLog ClassRefreshLog
	err := r.db.WithContext(ctx).Table(ClassRefreshLogTableName).
		Where("id = ?", logID).First(&refreshLog).Error
	if err != nil {
		return nil, err
	}
	return ClassRefreshLogDOToBO(&refreshLog), nil
}

func (r *RefreshLogRepo) createRefreshLog(ctx context.Context, db *gorm.DB, refreshLog *ClassRefreshLog) error {
	return db.WithContext(ctx).Create(refreshLog).Error
}
