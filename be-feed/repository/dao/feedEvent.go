package dao

import (
	"context"
	"time"

	"github.com/asynccnu/ccnubox-be/be-feed/repository/model"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"gorm.io/gorm"
)

// FeedEventDAO 定义接口
type FeedEventDAO interface {
	SaveFeedEvent(ctx context.Context, event model.FeedEvent) error
	GetFeedEventById(ctx context.Context, Id int64) (*model.FeedEvent, error)
	GetFeedEventsByStudentId(ctx context.Context, studentId string) ([]model.FeedEvent, error)
	RemoveFeedEvent(ctx context.Context, studentId string, id int64, status string) error
	InsertFeedEventList(ctx context.Context, event []model.FeedEvent) ([]model.FeedEvent, error)
	InsertFeedEvent(ctx context.Context, event *model.FeedEvent) (*model.FeedEvent, error)
	InsertFeedEventListByTx(ctx context.Context, tx *gorm.DB, events []model.FeedEvent) ([]model.FeedEvent, error)
	BeginTx(ctx context.Context) (*gorm.DB, error)
}

type feedEventDAO struct {
	gorm *gorm.DB
}

func NewFeedEventDAO(db *gorm.DB) FeedEventDAO {
	return &feedEventDAO{gorm: db}
}

func (dao *feedEventDAO) SaveFeedEvent(ctx context.Context, event model.FeedEvent) error {
	err := dao.gorm.WithContext(ctx).Model(&model.FeedEvent{}).Where("id = ?", event.ID).Save(event).Error
	if err != nil {
		return errorx.Errorf("dao: save feed event failed, id: %d, err: %w", event.ID, err)
	}
	return nil
}

func (dao *feedEventDAO) GetFeedEventById(ctx context.Context, Id int64) (*model.FeedEvent, error) {
	d := model.FeedEvent{}
	err := dao.gorm.WithContext(ctx).Model(&model.FeedEvent{}).
		Where("id = ?", Id).
		First(&d).Error
	if err != nil {
		return nil, errorx.Errorf("dao: get feed event by id failed, id: %d, err: %w", Id, err)
	}
	return &d, nil
}

func (dao *feedEventDAO) GetFeedEventsByStudentId(ctx context.Context, studentId string) ([]model.FeedEvent, error) {
	var resp []model.FeedEvent
	err := dao.gorm.WithContext(ctx).
		Model(&model.FeedEvent{}).
		Where("student_id = ?", studentId).
		Order("created_at DESC").
		Limit(20).
		Find(&resp).Error
	if err != nil {
		return nil, errorx.Errorf("dao: get feed events by student_id failed, sid: %s, err: %w", studentId, err)
	}
	return resp, nil
}

func (dao *feedEventDAO) RemoveFeedEvent(ctx context.Context, studentId string, id int64, status string) error {
	query := dao.gorm.WithContext(ctx).Model(&model.FeedEvent{})
	if studentId != "" {
		query = query.Where("student_id = ?", studentId)
	}
	if id != 0 {
		query = query.Where("id = ?", id)
	}

	if status == "read" {
		query = query.Where("`read` = ?", true)
	} else if status == "all" {
		// all 状态下不加 read 过滤条件
	} else {
		query = query.Where("`read` = ?", false)
	}

	err := query.Update("deleted_at", time.Now()).Error
	if err != nil {
		return errorx.Errorf("dao: remove feed event failed, sid: %s, id: %d, status: %s, err: %w", studentId, id, status, err)
	}
	return nil
}

func (dao *feedEventDAO) InsertFeedEventList(ctx context.Context, events []model.FeedEvent) ([]model.FeedEvent, error) {
	now := time.Now().Unix()
	for i := range events {
		events[i].CreatedAt = now
		events[i].UpdatedAt = now
	}
	err := dao.gorm.WithContext(ctx).Model(&model.FeedEvent{}).CreateInBatches(events, 1000).Error
	if err != nil {
		return nil, errorx.Errorf("dao: batch insert feed events failed, count: %d, err: %w", len(events), err)
	}
	return events, nil
}

func (dao *feedEventDAO) InsertFeedEvent(ctx context.Context, event *model.FeedEvent) (*model.FeedEvent, error) {
	now := time.Now().Unix()
	event.CreatedAt = now
	event.UpdatedAt = now
	err := dao.gorm.WithContext(ctx).Model(&model.FeedEvent{}).Create(event).Error
	if err != nil {
		return nil, errorx.Errorf("dao: insert single feed event failed, sid: %s, err: %w", event.StudentId, err)
	}
	return event, nil
}

func (dao *feedEventDAO) InsertFeedEventListByTx(ctx context.Context, tx *gorm.DB, events []model.FeedEvent) ([]model.FeedEvent, error) {
	now := time.Now().Unix()
	for i := range events {
		events[i].CreatedAt = now
		events[i].UpdatedAt = now
	}
	err := tx.WithContext(ctx).Model(&model.FeedEvent{}).CreateInBatches(events, 1000).Error
	if err != nil {
		return nil, errorx.Errorf("dao: tx batch insert feed events failed, count: %d, err: %w", len(events), err)
	}
	return events, nil
}

func (dao *feedEventDAO) BeginTx(ctx context.Context) (*gorm.DB, error) {
	tx := dao.gorm.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, errorx.Errorf("dao: begin transaction failed, err: %w", tx.Error)
	}
	return tx, nil
}
