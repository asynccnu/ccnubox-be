package data

import (
	"context"
	"errors"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/conf"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/model"
	"gorm.io/gorm"
	"time"
)

type RefreshLogRepo struct {
	db              *gorm.DB
	refreshInterval time.Duration // 刷新间隔,当前时间距离上次刷新时间超过该值时,需要重新刷新
}

func NewRefreshLogRepo(db *gorm.DB, cf *conf.Server) *RefreshLogRepo {
	refreshInterval := time.Minute
	if cf.RefreshInterval > 0 {
		refreshInterval = time.Duration(cf.RefreshInterval) * time.Second
	}
	return &RefreshLogRepo{
		db:              db,
		refreshInterval: refreshInterval,
	}
}

// InsertRefreshLog 插入一条刷新记录
func (r *RefreshLogRepo) InsertRefreshLog(ctx context.Context, stuID, year, semester string) (uint64, error) {

	refreshLog := model.ClassRefreshLog{
		StuID:     stuID,
		Year:      year,
		Semester:  semester,
		Status:    model.Pending,
		UpdatedAt: time.Now(),
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		//先检查下stuID-year-semester是否存在
		//如果不存在,则插入
		//如果存在的记录的更新时间距离当前时间超过刷新间隔,则新建一条记录
		//如果存在的记录的更新时间距离当前时间未超过刷新间隔,如果记录的状态是Pending或Ready,则返回错误
		//如果存在的记录的状态是Failed,则新建一条记录
		var records struct {
			Status    string
			UpdatedAt time.Time
		}

		err := tx.Table(model.ClassRefreshLogTableName).Select("status,updated_at").
			Where("stu_id = ? and year = ? and semester = ?", stuID, year, semester).
			Order("updated_at desc").
			First(&records).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.createRefreshLog(ctx, tx, &refreshLog)
		}
		if err != nil {
			return err
		}
		if records.UpdatedAt.Before(refreshLog.UpdatedAt.Add(-r.refreshInterval)) {
			return r.createRefreshLog(ctx, tx, &refreshLog)
		}
		if records.Status == model.Failed {
			return r.createRefreshLog(ctx, tx, &refreshLog)
		}
		return errors.New("there are pending or ready records recently")
	})

	if err != nil {
		return 0, err
	}
	return refreshLog.ID, nil
}

func (r *RefreshLogRepo) UpdateRefreshLogStatus(ctx context.Context, logID uint64, status string) error {
	return r.db.WithContext(ctx).Table(model.ClassRefreshLogTableName).
		Where("id = ?", logID).Update("status", status).Error
}

// SearchRefreshLog 查找在refreshInterval时间内的最新的一条记录
func (r *RefreshLogRepo) SearchRefreshLog(ctx context.Context, stuID, year, semester string) (*model.ClassRefreshLog, error) {
	var refreshLog model.ClassRefreshLog
	err := r.db.WithContext(ctx).Table(model.ClassRefreshLogTableName).
		Where("stu_id = ? and year = ? and semester = ? and updated_at > ?", stuID, year, semester, time.Now().Add(-r.refreshInterval)).
		Order("updated_at desc").First(&refreshLog).Error
	if err != nil {
		return nil, err
	}
	return &refreshLog, nil
}

// GetLastRefreshTime 返回最后一次刷新成功的时间
func (r *RefreshLogRepo) GetLastRefreshTime(ctx context.Context, stuID, year, semester string, beforeTime time.Time) *time.Time {
	var refreshLog model.ClassRefreshLog
	err := r.db.WithContext(ctx).Table(model.ClassRefreshLogTableName).
		Where("stu_id = ? and year = ? and semester = ? and updated_at < ? and status = ?", stuID, year, semester, beforeTime, model.Ready).
		Order("updated_at desc").First(&refreshLog).Error
	if err != nil {
		return nil
	}
	return &refreshLog.UpdatedAt
}

// GetRefreshLogByID  查找指定ID的记录
func (r *RefreshLogRepo) GetRefreshLogByID(ctx context.Context, logID uint64) (*model.ClassRefreshLog, error) {
	var refreshLog model.ClassRefreshLog
	err := r.db.WithContext(ctx).Table(model.ClassRefreshLogTableName).
		Where("id = ?", logID).First(&refreshLog).Error
	if err != nil {
		return nil, err
	}
	return &refreshLog, nil
}

func (r *RefreshLogRepo) createRefreshLog(ctx context.Context, db *gorm.DB, refreshLog *model.ClassRefreshLog) error {
	return db.WithContext(ctx).Create(refreshLog).Error
}
