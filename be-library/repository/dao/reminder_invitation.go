package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

func (d *ReminderDAO) Invitations(ctx context.Context, teamID string, recipients []string, forUpdate bool) ([]LibraryTeamInvitation, error) {
	var rows []LibraryTeamInvitation
	query := d.db.WithContext(ctx).Where("team_id = ? AND recipient_student_id IN ?", teamID, recipients).Order("recipient_student_id ASC")
	if forUpdate {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	err := query.Find(&rows).Error
	return rows, err
}

// 调用方持有收件人订阅行锁，额度统计和新事实插入必须在同一事务内。
func (d *ReminderDAO) InvitationDailyCount(ctx context.Context, recipient string, start, end time.Time) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&LibraryTeamInvitation{}).
		Where("recipient_student_id = ? AND disposition = ? AND created_at >= ? AND created_at < ?", recipient, "enqueued", start, end).Count(&count).Error
	return count, err
}

// 唯一键竞争由调用方回滚并重试整个事务，不能更新胜出事实的时间或载荷。
func (d *ReminderDAO) CreateInvitation(ctx context.Context, row *LibraryTeamInvitation, outbox *NotificationOutbox) error {
	if err := d.db.WithContext(ctx).Create(row).Error; err != nil {
		return err
	}
	if outbox == nil {
		return nil
	}
	return d.db.WithContext(ctx).Create(outbox).Error
}

// InvitationDAO 避免 GORM 在慢查询或唯一键竞争时将邀请数组、学号和载荷打印到日志。
// 错误仍返回调用方，由固定结果指标观测，不影响其他提醒的 SQL 日志配置。
func (d *ReminderDAO) InvitationDAO() *ReminderDAO {
	return &ReminderDAO{db: d.db.Session(&gorm.Session{Logger: logger.Discard})}
}
