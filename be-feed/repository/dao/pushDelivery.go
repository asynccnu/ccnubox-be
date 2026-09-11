package dao

import (
	"context"
	"errors"
	"time"

	"github.com/asynccnu/ccnubox-be/be-feed/repository/model"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"gorm.io/gorm"
)

var ErrFeedEventNotFound = errors.New("feed event not found")

type PushDeliveryDAO interface {
	RecoverSending(ctx context.Context, before int64) error
	ListDue(ctx context.Context, now int64, limit int) ([]model.FeedPushDelivery, error)
	Claim(ctx context.Context, id int64) (bool, error)
	GetFeedEvent(ctx context.Context, id int64) (*model.FeedEvent, error)
	SaveCID(ctx context.Context, id int64, cid string) error
	MarkSent(ctx context.Context, id int64) error
	MarkSuppressed(ctx context.Context, id int64) error
	MarkRetry(ctx context.Context, id int64, attempts int, nextAttemptAt int64, lastError string, failed bool) error
}

type pushDeliveryDAO struct {
	db *gorm.DB
}

func NewPushDeliveryDAO(db *gorm.DB) PushDeliveryDAO {
	return &pushDeliveryDAO{db: db}
}

// RecoverSending 只恢复 updated_at 早于 before 的 sending 记录，
// 避免把仍在推送中的记录误重置为 pending 导致重复投递。
func (d *pushDeliveryDAO) RecoverSending(ctx context.Context, before int64) error {
	if err := d.db.WithContext(ctx).Model(&model.FeedPushDelivery{}).
		Where("status = ? AND updated_at < ?", model.PushDeliverySending, before).
		Update("status", model.PushDeliveryPending).Error; err != nil {
		return errorx.Errorf("dao: recover sending push deliveries failed: %w", err)
	}
	return nil
}

func (d *pushDeliveryDAO) ListDue(ctx context.Context, now int64, limit int) ([]model.FeedPushDelivery, error) {
	var deliveries []model.FeedPushDelivery
	err := d.db.WithContext(ctx).
		Where("status = ? AND next_attempt_at <= ?", model.PushDeliveryPending, now).
		Order("priority DESC, next_attempt_at ASC, id ASC").
		Limit(limit).
		Find(&deliveries).Error
	if err != nil {
		return nil, errorx.Errorf("dao: list due push deliveries failed: %w", err)
	}
	return deliveries, nil
}

func (d *pushDeliveryDAO) Claim(ctx context.Context, id int64) (bool, error) {
	result := d.db.WithContext(ctx).Model(&model.FeedPushDelivery{}).
		Where("id = ? AND status = ?", id, model.PushDeliveryPending).
		Update("status", model.PushDeliverySending)
	if result.Error != nil {
		return false, errorx.Errorf("dao: claim push delivery failed, id: %d, err: %w", id, result.Error)
	}
	return result.RowsAffected == 1, nil
}

func (d *pushDeliveryDAO) GetFeedEvent(ctx context.Context, id int64) (*model.FeedEvent, error) {
	var event model.FeedEvent
	if err := d.db.WithContext(ctx).First(&event, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorx.Errorf("dao: push feed event not found, id: %d, err: %w", id, ErrFeedEventNotFound)
		}
		return nil, errorx.Errorf("dao: get push feed event failed, id: %d, err: %w", id, err)
	}
	return &event, nil
}

func (d *pushDeliveryDAO) SaveCID(ctx context.Context, id int64, cid string) error {
	result := d.db.WithContext(ctx).Model(&model.FeedPushDelivery{}).
		Where("id = ? AND status = ? AND cid = ?", id, model.PushDeliverySending, "").
		Update("cid", cid)
	if result.Error != nil {
		return errorx.Errorf("dao: save push delivery cid failed, id: %d, err: %w", id, result.Error)
	}
	if result.RowsAffected != 1 {
		return errorx.Errorf("dao: push delivery cid cannot be saved, id: %d", id)
	}
	return nil
}

func (d *pushDeliveryDAO) MarkSent(ctx context.Context, id int64) error {
	return d.markFinal(ctx, id, model.PushDeliverySent)
}

func (d *pushDeliveryDAO) MarkSuppressed(ctx context.Context, id int64) error {
	return d.markFinal(ctx, id, model.PushDeliverySuppressed)
}

func (d *pushDeliveryDAO) markFinal(ctx context.Context, id int64, status string) error {
	result := d.db.WithContext(ctx).Model(&model.FeedPushDelivery{}).
		Where("id = ? AND status = ?", id, model.PushDeliverySending).
		Updates(map[string]any{
			"status":     status,
			"last_error": "",
		})
	if result.Error != nil {
		return errorx.Errorf("dao: mark push delivery final failed, id: %d, err: %w", id, result.Error)
	}
	if result.RowsAffected != 1 {
		return errorx.Errorf("dao: push delivery is not sending, id: %d", id)
	}
	return nil
}

func (d *pushDeliveryDAO) MarkRetry(ctx context.Context, id int64, attempts int, nextAttemptAt int64, lastError string, failed bool) error {
	status := model.PushDeliveryPending
	if failed {
		status = model.PushDeliveryFailed
	}
	result := d.db.WithContext(ctx).Model(&model.FeedPushDelivery{}).
		Where("id = ? AND status = ?", id, model.PushDeliverySending).
		Updates(map[string]any{
			"status":          status,
			"attempts":        attempts,
			"next_attempt_at": nextAttemptAt,
			"last_error":      lastError,
			"updated_at":      time.Now().Unix(),
		})
	if result.Error != nil {
		return errorx.Errorf("dao: retry push delivery failed, id: %d, err: %w", id, result.Error)
	}
	if result.RowsAffected != 1 {
		return errorx.Errorf("dao: push delivery is not sending, id: %d", id)
	}
	return nil
}
