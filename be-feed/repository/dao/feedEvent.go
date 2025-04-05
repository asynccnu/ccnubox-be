package dao

import (
	"context"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/model"
	"gorm.io/gorm"
	"time"
)

// feedEvent由于使用了表名进行查询,gorm的自动处理时间的作用将失效
type FeedEventDAO interface {
	// 上部分是用于对 index 进行处理,下部分是对具体的 feedEvent 进行处理
	SaveFeedEvent(ctx context.Context, event model.FeedEvent) error
	GetFeedEventById(ctx context.Context, Id int64) (*model.FeedEvent, error)
	GetFeedEventsByStudentId(ctx context.Context, studentId string) ([]model.FeedEvent, error)
	RemoveFeedEvent(ctx context.Context, studentId string, id int64, status string) error
	InsertFeedEventList(ctx context.Context, event []model.FeedEvent) ([]model.FeedEvent, error)
	InsertFeedEvent(ctx context.Context, event *model.FeedEvent) (*model.FeedEvent, error)
	InsertFeedEventListByTx(ctx context.Context, tx *gorm.DB, events []model.FeedEvent) ([]model.FeedEvent, error)
	// 用于事务
	BeginTx(ctx context.Context) (*gorm.DB, error)
}

type feedEventDAO struct {
	gorm *gorm.DB
}

func NewFeedEventDAO(db *gorm.DB) FeedEventDAO {
	return &feedEventDAO{gorm: db}
}

func (dao *feedEventDAO) SaveFeedEvent(ctx context.Context, event model.FeedEvent) error {
	return dao.gorm.WithContext(ctx).Model(&model.FeedEvent{}).Where("id = ?", event.ID).Save(event).Error
}

// GetFeedEventById 获取指定 ID 的 FeedEvent
func (dao *feedEventDAO) GetFeedEventById(ctx context.Context, Id int64) (*model.FeedEvent, error) {
	d := model.FeedEvent{}
	err := dao.gorm.WithContext(ctx).Model(&model.FeedEvent{}).
		Where("id = ?", Id). // 过滤已软删除的记录
		First(&d).Error
	return &d, err
}

// GetFeedEventsByStudentId 获取指定 StudentId 的 FeedEvent 列表
func (dao *feedEventDAO) GetFeedEventsByStudentId(ctx context.Context, studentId string) ([]model.FeedEvent, error) {
	var resp []model.FeedEvent
	err := dao.gorm.WithContext(ctx).
		Model(&model.FeedEvent{}).
		Where("student_id = ?", studentId). // 过滤已软删除的记录
		Find(&resp).
		Order("created_at DESC"). // 按照 created_at 字段降序排列
		Limit(20).
		Error
	return resp, err
}

// DelFeedEventById 软删除指定 ID 的 FeedEvent
func (dao *feedEventDAO) RemoveFeedEvent(ctx context.Context, studentId string, id int64, status string) error {
	query := dao.gorm.WithContext(ctx).Model(&model.FeedEvent{})
	if studentId != "" {
		query = query.Where("student_id = ?", studentId)
	}
	if id != 0 {
		query = query.Where("id = ?", id)
	}

	if status == "read" {
		query = query.Where("read = ?", true)
	} else if status == "all" {
		//省略操作
	} else {
		query = query.Where("read = ?", false)
	}

	return query.Update("deleted_at", time.Now()).Error
}

// DelFeedEventsByStudentId 软删除指定 StudentId 的 FeedEvent
func (dao *feedEventDAO) DelFeedEventsByStudentId(ctx context.Context, studentId string) error {
	return dao.gorm.WithContext(ctx).Model(&model.FeedEvent{}).
		Where("student_id = ?", studentId).
		Update("deleted_at", time.Now().Unix()). // 设置软删除时间
		Error
}

// InsertFeedEventList 批量插入 FeedEvent，最多一次插入 1000 条
func (dao *feedEventDAO) InsertFeedEventList(ctx context.Context, events []model.FeedEvent) ([]model.FeedEvent, error) {
	now := time.Now().Unix()
	// 为每个事件设置时间戳
	for i := range events {
		events[i].CreatedAt = now
		events[i].UpdatedAt = now
	}
	err := dao.gorm.WithContext(ctx).Model(&model.FeedEvent{}).CreateInBatches(events, 1000).Error
	return events, err
}

// InsertFeedEvent 插入单个 FeedEvent
func (dao *feedEventDAO) InsertFeedEvent(ctx context.Context, event *model.FeedEvent) (*model.FeedEvent, error) {
	now := time.Now().Unix()
	event.CreatedAt = now
	event.UpdatedAt = now
	err := dao.gorm.WithContext(ctx).Model(&model.FeedEvent{}).Create(event).Error
	return event, err
}

// InsertFeedEventListByTx 使用事务批量插入 FeedEvent，最多一次插入 1000 条
func (dao *feedEventDAO) InsertFeedEventListByTx(ctx context.Context, tx *gorm.DB, events []model.FeedEvent) ([]model.FeedEvent, error) {
	now := time.Now().Unix()
	// 为每个事件设置时间戳
	for i := range events {
		events[i].CreatedAt = now
		events[i].UpdatedAt = now
	}
	err := tx.WithContext(ctx).Model(&model.FeedEvent{}).CreateInBatches(events, 1000).Error
	return events, err
}

// BeginTx 开启事务，用于批量插入或其他操作
func (dao *feedEventDAO) BeginTx(ctx context.Context) (*gorm.DB, error) {
	tx := dao.gorm.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	return tx, nil
}
