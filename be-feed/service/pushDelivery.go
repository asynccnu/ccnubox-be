package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/asynccnu/ccnubox-be/be-feed/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/model"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/pkg/metricsx"
)

const (
	pushDeliveryBatchSize      = 100
	pushDeliveryMaxPerDispatch = 2000
	pushDeliveryMaxConcurrency = 10
	pushDeliveryMaxAttempts    = 10
	pushDeliveryMaxBackoff     = time.Second << pushDeliveryMaxAttempts
	pushDeliveryErrorLimit     = 2048
	pushDeliveryStateTimeout   = 5 * time.Second
	// pushDeliveryRecoverTimeout：sending 超过该时长才视为孤儿记录并恢复。
	// 需大于单次投递的最长耗时（JPush HTTP 超时 15s + PreparePush/取 CID 等开销 + 状态回写窗口 5s）。
	pushDeliveryRecoverTimeout = time.Minute
)

type PushDeliveryService interface {
	RecoverSending(ctx context.Context) error
	DispatchDue(ctx context.Context) error
}

type pushDeliveryService struct {
	dao     dao.PushDeliveryDAO
	gate    dao.FeedUserConfigDAO
	push    PushService
	log     logger.Logger
	metrics *metricsx.FeedDeliveryMetrics
}

func (s *pushDeliveryService) RecoverSending(ctx context.Context) error {
	// 只恢复超过超时时长的 sending，避免并发调用方误恢复彼此在途的记录。
	before := time.Now().Add(-pushDeliveryRecoverTimeout).Unix()
	return s.dao.RecoverSending(ctx, before)
}

func NewPushDeliveryService(deliveryDAO dao.PushDeliveryDAO, gate dao.FeedUserConfigDAO, push PushService, metricSet *metricsx.Metrics, log logger.Logger) PushDeliveryService {
	var deliveryMetrics *metricsx.FeedDeliveryMetrics
	if metricSet != nil {
		deliveryMetrics = metricSet.Feed
	}
	return &pushDeliveryService{
		dao:     deliveryDAO,
		gate:    gate,
		push:    push,
		log:     log,
		metrics: deliveryMetrics,
	}
}

func (s *pushDeliveryService) DispatchDue(ctx context.Context) error {
	// 定时恢复遗留的 sending。控制器不会重叠调用 DispatchDue，
	// 因此单次调用内部的并发 worker 不会被这里误恢复；
	// 超时条件进一步保证即使将来出现并发调用方，也只恢复真正卡死的记录。
	if err := s.RecoverSending(ctx); err != nil {
		return err
	}

	processed := 0
	for processed < pushDeliveryMaxPerDispatch {
		if err := ctx.Err(); err != nil {
			return err
		}
		limit := min(pushDeliveryBatchSize, pushDeliveryMaxPerDispatch-processed)
		deliveries, err := s.dao.ListDue(ctx, time.Now().Unix(), limit)
		if err != nil {
			return err
		}
		if len(deliveries) == 0 {
			return nil
		}
		if err = s.dispatchBatch(ctx, deliveries); err != nil {
			return err
		}
		processed += len(deliveries)
		if len(deliveries) < limit {
			return nil
		}
	}
	return nil
}

func (s *pushDeliveryService) dispatchBatch(ctx context.Context, deliveries []model.FeedPushDelivery) error {
	workerCount := min(pushDeliveryMaxConcurrency, len(deliveries))
	jobs := make(chan model.FeedPushDelivery)
	errs := make(chan error, len(deliveries)+1)
	var wg sync.WaitGroup
	for range workerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for delivery := range jobs {
				if err := s.dispatchOne(ctx, delivery); err != nil {
					errs <- err
				}
			}
		}()
	}

sendLoop:
	for i := range deliveries {
		select {
		case jobs <- deliveries[i]:
		case <-ctx.Done():
			break sendLoop
		}
	}
	close(jobs)
	wg.Wait()
	if err := ctx.Err(); err != nil {
		errs <- err
	}
	close(errs)

	var batchErrors []error
	for err := range errs {
		batchErrors = append(batchErrors, err)
	}
	return errors.Join(batchErrors...)
}

func (s *pushDeliveryService) dispatchOne(ctx context.Context, delivery model.FeedPushDelivery) error {
	claimed, err := s.dao.Claim(ctx, delivery.ID)
	if err != nil || !claimed {
		return err
	}

	event, err := s.dao.GetFeedEvent(ctx, delivery.FeedEventID)
	if errors.Is(err, dao.ErrFeedEventNotFound) {
		err = s.markSuppressed(ctx, delivery.ID)
		if err == nil {
			if s.metrics != nil {
				s.metrics.PushDeliveryTotal.WithLabelValues("suppressed_missing_event").Inc()
			}
			return nil
		}
	}
	if err == nil && libraryPushExpired(event, time.Now()) {
		err = s.markExpired(ctx, delivery.ID)
		if err == nil {
			return nil
		}
	}
	if err == nil && strings.EqualFold(event.Type, "library") && s.gate != nil {
		enabled, gateErr := s.gate.IsLibraryEnabled(ctx, event.StudentId)
		if gateErr != nil {
			err = gateErr
		} else if !enabled {
			err = s.markSuppressed(ctx, delivery.ID)
			if err == nil {
				if s.metrics != nil {
					s.metrics.PushDeliveryTotal.WithLabelValues("suppressed_by_allow_list").Inc()
				}
				return nil
			}
		}
	}
	if err == nil {
		domainEvent := convFeedEventsFromModelToDomain([]model.FeedEvent{*event})[0]
		prepared, prepareErr := s.push.PreparePush(ctx, &domainEvent)
		err = prepareErr
		if err == nil && prepared == nil {
			err = s.markSuppressed(ctx, delivery.ID)
			if err == nil {
				if s.metrics != nil {
					s.metrics.PushDeliveryTotal.WithLabelValues("suppressed_no_target_or_disabled").Inc()
				}
				return nil
			}
		}
		if err == nil {
			cid := delivery.CID
			if cid == "" {
				cid, err = s.push.GetPushCID(ctx)
				if err == nil {
					err = s.saveCID(ctx, delivery.ID, cid)
				}
			}
			if err == nil && libraryPushExpired(event, time.Now()) {
				// 取目标、获取及保存 CID 也可能跨过有效期，实际推送前必须复核。
				err = s.markExpired(ctx, delivery.ID)
				if err == nil {
					return nil
				}
			}
			if err == nil {
				// 持久化 JPush 签发的 CID，让重试保持幂等。
				err = s.push.PushPreparedMSGWithCID(ctx, &domainEvent, prepared, cid)
			}
		}
	}
	if err == nil {
		err = s.markSent(ctx, delivery.ID)
		if err == nil {
			if s.metrics != nil {
				s.metrics.PushDeliveryTotal.WithLabelValues("sent").Inc()
			}
			return nil
		}
	}

	attempts := delivery.Attempts + 1
	backoff := time.Second << min(attempts, pushDeliveryMaxAttempts)
	if backoff > pushDeliveryMaxBackoff {
		backoff = pushDeliveryMaxBackoff
	}
	lastError := boundedPushError(err)
	failed := attempts >= pushDeliveryMaxAttempts
	stateCtx, cancel := pushDeliveryStateContext(ctx)
	markErr := s.dao.MarkRetry(stateCtx, delivery.ID, attempts, time.Now().Add(backoff).Unix(), lastError, failed)
	cancel()
	if markErr != nil {
		return markErr
	}
	if s.metrics != nil {
		result := "retry"
		if failed {
			result = "failed"
		}
		s.metrics.PushDeliveryTotal.WithLabelValues(result).Inc()
	}
	s.log.Warn("push delivery failed",
		logger.Int64("delivery_id", delivery.ID),
		logger.Int("attempt", attempts),
		logger.String("status", map[bool]string{true: "failed", false: "pending"}[failed]),
		logger.Error(err))
	return nil
}

// 时效性提醒以业务时间为截止点；缺失或非法时间不能当作永久有效。
// 普通通知和事实类图书馆消息不受此限制。
func libraryPushExpired(event *model.FeedEvent, now time.Time) bool {
	if !strings.EqualFold(event.Type, "library") {
		return false
	}
	var field string
	switch strings.ToUpper(strings.TrimSpace(event.ExtendFields["notification_type"])) {
	case "START_30":
		field = "start_at"
	case "END_10", "AWAY_60", "AWAY_80":
		field = "end_at"
	default:
		return false
	}
	expiresAt, err := strconv.ParseInt(event.ExtendFields[field], 10, 64)
	return err != nil || expiresAt <= 0 || expiresAt <= now.Unix()
}

func (s *pushDeliveryService) markExpired(ctx context.Context, id int64) error {
	err := s.markSuppressed(ctx, id)
	if err == nil && s.metrics != nil {
		s.metrics.PushDeliveryTotal.WithLabelValues("suppressed_expired").Inc()
	}
	return err
}

func (s *pushDeliveryService) saveCID(ctx context.Context, id int64, cid string) error {
	stateCtx, cancel := pushDeliveryStateContext(ctx)
	defer cancel()
	return s.dao.SaveCID(stateCtx, id, cid)
}

func (s *pushDeliveryService) markSent(ctx context.Context, id int64) error {
	stateCtx, cancel := pushDeliveryStateContext(ctx)
	defer cancel()
	return s.dao.MarkSent(stateCtx, id)
}

func (s *pushDeliveryService) markSuppressed(ctx context.Context, id int64) error {
	stateCtx, cancel := pushDeliveryStateContext(ctx)
	defer cancel()
	return s.dao.MarkSuppressed(stateCtx, id)
}

// pushDeliveryStateContext 确保批次超时后仍有短暂窗口回写已 claim 的记录状态。
func pushDeliveryStateContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), pushDeliveryStateTimeout)
}

// boundedPushError 限制错误文本长度，避免被包装的客户端错误包含大量响应内容。
// 数据库字段仅用于诊断，不得存储响应载荷。
func boundedPushError(err error) string {
	if err == nil {
		return ""
	}
	// 先替换非法 UTF-8 字节再沿字符边界截断：按字节硬截断可能切开多字节字符，
	// 非法序列会让 last_error 写入 MySQL 失败，记录卡在 sending 并在恢复后持续阻断后续投递。
	text := strings.TrimSpace(strings.ToValidUTF8(err.Error(), string(utf8.RuneError)))
	return truncateUTF8(text, pushDeliveryErrorLimit)
}

// truncateUTF8 在不超过 limit 字节的前提下沿 UTF-8 字符边界截断。
func truncateUTF8(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	for limit > 0 && !utf8.RuneStart(text[limit]) {
		limit--
	}
	return text[:limit]
}
