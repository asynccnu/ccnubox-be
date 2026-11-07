package dao

import (
	"context"
	"errors"
	"time"

	"github.com/asynccnu/ccnubox-be/be-grade/domain"
	"github.com/asynccnu/ccnubox-be/be-grade/repository/model"
	"gorm.io/gorm"
)

type NotificationStats struct {
	Pending int64
	Invalid int64
	Oldest  int64
}

type NotificationDAO interface {
	FindDue(ctx context.Context, now int64, limit int) ([]model.GradeNotificationOutbox, error)
	MarkPublished(ctx context.Context, id int64) error
	MarkFailed(ctx context.Context, id int64, attempts int64, next int64, invalid bool, summary string) error
	Stats(ctx context.Context) (NotificationStats, error)
}

// RequeueInvalidGradeNotification 仅供受控运维命令使用，校验成功后恢复原事件，不重建 key。
func RequeueInvalidGradeNotification(ctx context.Context, db *gorm.DB, id int64) error {
	var row model.GradeNotificationOutbox
	if err := db.WithContext(ctx).First(&row, "id = ? AND status = ?", id, model.NotificationInvalid).Error; err != nil {
		return err
	}
	if _, err := domain.ParseGradeNotification(row); err != nil {
		return err
	}
	result := db.WithContext(ctx).Model(&model.GradeNotificationOutbox{}).
		Where("id = ? AND status = ? AND payload = ?", id, model.NotificationInvalid, row.Payload).
		Updates(map[string]any{"status": model.NotificationPending, "next_attempt_at": time.Now().Unix(), "last_error": ""})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("grade notification changed during recovery")
	}
	return nil
}

type notificationDAO struct{ db *gorm.DB }

func NewNotificationDAO(db *gorm.DB) NotificationDAO { return &notificationDAO{db: db} }

func (d *notificationDAO) FindDue(ctx context.Context, now int64, limit int) ([]model.GradeNotificationOutbox, error) {
	var rows []model.GradeNotificationOutbox
	err := d.db.WithContext(ctx).Where("status = ? AND next_attempt_at <= ?", model.NotificationPending, now).
		Order("next_attempt_at, id").Limit(limit).Find(&rows).Error
	return rows, err
}

func (d *notificationDAO) MarkPublished(ctx context.Context, id int64) error {
	return d.db.WithContext(ctx).Model(&model.GradeNotificationOutbox{}).Where("id = ? AND status = ?", id, model.NotificationPending).
		Updates(map[string]any{"status": model.NotificationPublished, "attempts": gorm.Expr("attempts + 1"), "last_error": "", "updated_at": time.Now().Unix()}).Error
}

func (d *notificationDAO) MarkFailed(ctx context.Context, id int64, attempts int64, next int64, invalid bool, summary string) error {
	state := model.NotificationPending
	if invalid {
		state = model.NotificationInvalid
	}
	if len([]rune(summary)) > 512 {
		summary = string([]rune(summary)[:512])
	}
	return d.db.WithContext(ctx).Model(&model.GradeNotificationOutbox{}).Where("id = ? AND status = ?", id, model.NotificationPending).
		Updates(map[string]any{"status": state, "attempts": attempts, "next_attempt_at": next, "last_error": summary, "updated_at": time.Now().Unix()}).Error
}

func (d *notificationDAO) Stats(ctx context.Context) (NotificationStats, error) {
	var stats NotificationStats
	query := func() *gorm.DB { return d.db.WithContext(ctx).Model(&model.GradeNotificationOutbox{}) }
	if err := query().Where("status = ?", model.NotificationPending).Count(&stats.Pending).Error; err != nil {
		return stats, err
	}
	if err := query().Where("status = ?", model.NotificationInvalid).Count(&stats.Invalid).Error; err != nil {
		return stats, err
	}
	err := query().Select("COALESCE(MIN(created_at), 0)").Where("status = ?", model.NotificationPending).Scan(&stats.Oldest).Error
	return stats, err
}
