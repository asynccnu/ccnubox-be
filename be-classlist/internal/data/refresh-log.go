package data

import (
	"context"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/data/do"
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
func (r *RefreshLogRepo) InsertRefreshLog(ctx context.Context, stuID, year, semester string, logTime time.Time) (uint64, error) {
	refreshLog := do.ClassRefreshLog{
		StuID:     stuID,
		Year:      year,
		Semester:  semester,
		Status:    do.Pending,
		UpdatedAt: logTime,
	}
	err := r.createRefreshLog(ctx, r.db, &refreshLog)
	if err != nil {
		return 0, err
	}
	return refreshLog.ID, nil
}

func (r *RefreshLogRepo) UpdateRefreshLogStatus(ctx context.Context, logID uint64, status string) error {
	return r.db.WithContext(ctx).Table(do.ClassRefreshLogTableName).
		Where("id = ?", logID).Update("status", status).Error
}

// SearchNewestRefreshLog 查找在指定时间内的最新的一条记录
func (r *RefreshLogRepo) SearchNewestRefreshLog(ctx context.Context, stuID, year, semester string, endTime time.Time) (*do.ClassRefreshLog, error) {
	var refreshLog do.ClassRefreshLog
	err := r.db.WithContext(ctx).Table(do.ClassRefreshLogTableName).
		Where("stu_id = ? and year = ? and semester = ? and updated_at < ?", stuID, year, semester, endTime).
		Order("updated_at desc").First(&refreshLog).Error
	if err != nil {
		return nil, err
	}
	return &refreshLog, nil
}

// GetLastRefreshTime 返回最后一次刷新成功的时间
func (r *RefreshLogRepo) GetLastRefreshTime(ctx context.Context, stuID, year, semester string, beforeTime time.Time) *time.Time {
	var refreshLog do.ClassRefreshLog
	err := r.db.WithContext(ctx).Table(do.ClassRefreshLogTableName).
		Where("stu_id = ? and year = ? and semester = ? and updated_at < ? and status = ?", stuID, year, semester, beforeTime, do.Ready).
		Order("updated_at desc").First(&refreshLog).Error
	if err != nil {
		return nil
	}
	return &refreshLog.UpdatedAt
}

// GetRefreshLogByID  查找指定ID的记录
func (r *RefreshLogRepo) GetRefreshLogByID(ctx context.Context, logID uint64) (*do.ClassRefreshLog, error) {
	var refreshLog do.ClassRefreshLog
	err := r.db.WithContext(ctx).Table(do.ClassRefreshLogTableName).
		Where("id = ?", logID).First(&refreshLog).Error
	if err != nil {
		return nil, err
	}
	return &refreshLog, nil
}

// DeleteRedundantLogs 删除冗余的刷新记录
func (r *RefreshLogRepo) DeleteRedundantLogs(ctx context.Context, stuID, year, semester string) error {

	var ids []int

	// 首先找到所有成功的记录的ID
	err := r.db.WithContext(ctx).Table(do.ClassRefreshLogTableName).
		Where("stu_id = ? AND year = ? AND semester = ? AND status = ?",
			stuID, year, semester, do.Ready).Order("id DESC").Pluck("id", &ids).Error

	if err != nil {
		return err
	}

	// 如果没有找到成功的记录或者成功的记录等于1，直接返回
	if len(ids) <= 1 {
		return nil
	}

	// 获取除最新一条记录外的所有记录ID
	toDelete := make([]int, len(ids)-1)
	copy(toDelete, ids[1:])

	// 并添加已失败的刷新记录
	var failedLog []int
	err = r.db.WithContext(ctx).Table(do.ClassRefreshLogTableName).
		Where("stu_id = ? AND year = ? AND semester = ? AND status = ?",
			stuID, year, semester, do.Failed).Order("id DESC").Pluck("id", &failedLog).Error
	if err != nil {
		failedLog = nil
	}

	// 将失败的记录ID添加到待删除列表中
	if len(failedLog) > 0 {
		toDelete = append(toDelete, failedLog...)
	}
	// 删除这些记录
	if len(toDelete) > 0 {
		err = r.db.WithContext(ctx).Table(do.ClassRefreshLogTableName).
			Where("id IN ?", toDelete).Delete(&do.ClassRefreshLog{}).Error
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *RefreshLogRepo) createRefreshLog(ctx context.Context, db *gorm.DB, refreshLog *do.ClassRefreshLog) error {
	return db.WithContext(ctx).Create(refreshLog).Error
}
