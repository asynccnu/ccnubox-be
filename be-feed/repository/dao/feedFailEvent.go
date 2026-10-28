package dao

import (
	"context"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/model"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"gorm.io/gorm"
)

// FeedFailEventDAO 接口定义
type FeedFailEventDAO interface {
	GetFeedFailEventsByStudentId(ctx context.Context, studentId string) ([]model.FeedFailEvent, error)
	DelFeedFailEventsByStudentId(ctx context.Context, studentId string, ids []int64) error
	InsertFeedFailEventList(ctx context.Context, events []model.FeedFailEvent) error
}

type feedFailEventDAO struct {
	gorm *gorm.DB
}

func NewFeedFailEventDAO(db *gorm.DB) FeedFailEventDAO {
	return &feedFailEventDAO{gorm: db}
}

// GetFeedFailEventsByStudentId 获取指定 StudentId 的失败 FeedEvent 列表
func (dao *feedFailEventDAO) GetFeedFailEventsByStudentId(ctx context.Context, studentId string) ([]model.FeedFailEvent, error) {
	var resp []model.FeedFailEvent
	err := withFeedSubscription(dao.gorm.WithContext(ctx).Model(&model.FeedFailEvent{}), "feed_fail_events").
		Select("feed_fail_events.*").
		Where("feed_fail_events.student_id = ?", studentId).
		Find(&resp).Error
	if err != nil {
		return nil, errorx.Errorf("dao: get feed fail events failed, sid: %s, err: %w", studentId, err)
	}
	return resp, nil
}

// DelFeedFailEventsByStudentId 只消费本次读取的失败消息，仍保留当前订阅关闭的类型。
func (dao *feedFailEventDAO) DelFeedFailEventsByStudentId(ctx context.Context, studentId string, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	err := withFailFeedSubscription(dao.gorm.WithContext(ctx).Model(&model.FeedFailEvent{}), studentId).
		Where("student_id = ? AND id IN ?", studentId, ids).
		Delete(&model.FeedFailEvent{}).
		Error
	if err != nil {
		return errorx.Errorf("dao: delete feed fail events failed, sid: %s, err: %w", studentId, err)
	}
	return nil
}

// InsertFeedFailEventList 批量插入失败的 FeedEvent
func (dao *feedFailEventDAO) InsertFeedFailEventList(ctx context.Context, events []model.FeedFailEvent) error {
	if len(events) == 0 {
		return nil
	}
	err := dao.gorm.WithContext(ctx).Create(events).Error
	if err != nil {
		return errorx.Errorf("dao: batch insert feed fail events failed, count: %d, err: %w", len(events), err)
	}
	return nil
}
