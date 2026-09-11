package dao

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/asynccnu/ccnubox-be/be-feed/repository/model"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	MarkFeedEventRead(ctx context.Context, studentID string, id int64) error
	DedupeKeyExists(ctx context.Context, studentID, dedupeKey string) (bool, error)
	StoreFeedEvents(ctx context.Context, events []model.FeedEvent) (inserted []model.FeedEvent, suppressed int, err error)
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
		Order("created_at DESC, id DESC").
		Limit(20).
		Find(&resp).Error
	if err != nil {
		return nil, errorx.Errorf("dao: get feed events by student_id failed, sid: %s, err: %w", studentId, err)
	}
	return resp, nil
}

func (dao *feedEventDAO) MarkFeedEventRead(ctx context.Context, studentID string, id int64) error {
	result := dao.gorm.WithContext(ctx).
		Model(&model.FeedEvent{}).
		Where("id = ? AND student_id = ?", id, studentID).
		Update("read", true)
	if result.Error != nil {
		return errorx.Errorf("dao: mark feed event read failed, sid: %s, id: %d, err: %w", studentID, id, result.Error)
	}
	if result.RowsAffected == 0 {
		var count int64
		err := dao.gorm.WithContext(ctx).
			Model(&model.FeedEvent{}).
			Where("id = ? AND student_id = ?", id, studentID).
			Count(&count).Error
		if err != nil {
			return errorx.Errorf("dao: check feed event after marking read failed, sid: %s, id: %d, err: %w", studentID, id, err)
		}
		if count == 0 {
			return gorm.ErrRecordNotFound
		}
	}
	return nil
}

func (dao *feedEventDAO) DedupeKeyExists(ctx context.Context, studentID, dedupeKey string) (bool, error) {
	if studentID == "" || dedupeKey == "" {
		return false, nil
	}
	var count int64
	err := dao.gorm.WithContext(ctx).Unscoped().Model(&model.FeedEvent{}).
		Where("student_id = ? AND dedupe_key = ?", studentID, dedupeKey).
		Count(&count).Error
	if err != nil {
		return false, errorx.Errorf("dao: check feed dedupe key failed, err: %w", err)
	}
	return count > 0, nil
}

// StoreFeedEvents 封装事务，避免竞争
func (dao *feedEventDAO) StoreFeedEvents(ctx context.Context, events []model.FeedEvent) (inserted []model.FeedEvent, suppressed int, err error) {
	inserted = make([]model.FeedEvent, 0, len(events))
	err = dao.gorm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i := range events {
			event := events[i]
			if strings.EqualFold(event.Type, "library") {
				var config model.FeedUserConfig
				findErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
					Where("student_id = ?", event.StudentId).
					First(&config).Error
				if errors.Is(findErr, gorm.ErrRecordNotFound) {
					suppressed++
					continue
				}
				if findErr != nil {
					return findErr
				}
				if config.PushConfig&(1<<model.LibraryPos) == 0 {
					suppressed++
					continue
				}
			}

			result := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "student_id"}, {Name: "dedupe_key"}},
				DoNothing: true,
			}).Create(&event)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				continue
			}
			delivery := model.FeedPushDelivery{
				FeedEventID:   event.ID,
				StudentId:     event.StudentId,
				Status:        model.PushDeliveryPending,
				Priority:      pushDeliveryPriority(event),
				NextAttemptAt: time.Now().Unix(),
			}
			if err := tx.Create(&delivery).Error; err != nil {
				return err
			}
			inserted = append(inserted, event)
		}
		return nil
	})
	if err != nil {
		return nil, 0, errorx.Errorf("dao: store feed events transaction failed, err: %w", err)
	}
	return inserted, suppressed, nil
}

// 时效提醒优先于普通广播投递，避免全量消息占满先进先出队列。
func pushDeliveryPriority(event model.FeedEvent) int {
	if !strings.EqualFold(event.Type, "library") {
		return 0
	}
	switch strings.ToUpper(strings.TrimSpace(event.ExtendFields["notification_type"])) {
	case "START_30", "END_10", "AWAY_60", "AWAY_80":
		return 100
	default:
		return 0
	}
}

func (dao *feedEventDAO) RemoveFeedEvent(ctx context.Context, studentId string, id int64, status string) error {
	if strings.TrimSpace(studentId) == "" {
		return errorx.Errorf("dao: student id is required to remove feed events")
	}
	query := dao.gorm.WithContext(ctx).Model(&model.FeedEvent{}).Where("student_id = ?", studentId)
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
