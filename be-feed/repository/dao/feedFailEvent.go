package dao

import (
	"context"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/model"
	"gorm.io/gorm"
)

// feedEvent由于使用了表名进行查询,gorm的自动处理时间的作用将失效
type FeedFailEventDAO interface {
	GetFeedFailEventsByStudentId(ctx context.Context, studentId string) ([]model.FeedFailEvent, error)
	DelFeedFailEventsByStudentId(ctx context.Context, studentId string) error
	InsertFeedFailEventList(ctx context.Context, events []model.FeedFailEvent) error
}

type feedFailEventDAO struct {
	gorm *gorm.DB
}

func NewFeedFailEventDAO(db *gorm.DB) FeedFailEventDAO {
	return &feedFailEventDAO{gorm: db}
}

// GetFeedEventsByStudentId 获取指定 StudentId 的 FeedEvent 列表
func (dao *feedFailEventDAO) GetFeedFailEventsByStudentId(ctx context.Context, studentId string) ([]model.FeedFailEvent, error) {
	var resp []model.FeedFailEvent
	err := dao.gorm.WithContext(ctx).
		Where("student_id = ?", studentId).
		Find(&resp).Error
	return resp, err
}

// DelFeedEventsByStudentId 软删除指定 StudentId 的 FeedEvent
func (dao *feedFailEventDAO) DelFeedFailEventsByStudentId(ctx context.Context, studentId string) error {
	return dao.gorm.WithContext(ctx).
		Where("student_id = ?", studentId).
		Delete(&model.FeedFailEvent{}).
		Error
}

// InsertFeedEventList 批量插入 FeedEvent，最多一次插入 1000 条
func (dao *feedFailEventDAO) InsertFeedFailEventList(ctx context.Context, events []model.FeedFailEvent) error {
	return dao.gorm.WithContext(ctx).Create(events).Error
}
