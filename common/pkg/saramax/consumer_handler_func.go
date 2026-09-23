package saramax

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

type Handler[T any] struct {
	l   logger.Logger
	fn  func(context.Context, []*sarama.ConsumerMessage, []T) error
	cfg *HandlerConfig
}

func NewHandler[T any](
	l logger.Logger,
	cfg *HandlerConfig,
	fn func(t []T) error) *Handler[T] {
	return NewContextHandler(l, cfg, func(_ context.Context, events []T) error {
		return fn(events)
	})
}

// NewContextHandler 将消费会话的取消信号传递给业务处理。
func NewContextHandler[T any](
	l logger.Logger,
	cfg *HandlerConfig,
	fn func(context.Context, []T) error) *Handler[T] {
	return &Handler[T]{
		l: l,
		fn: func(ctx context.Context, _ []*sarama.ConsumerMessage, events []T) error {
			return fn(ctx, events)
		},
		cfg: cfg,
	}
}

func (h *Handler[T]) Setup(session sarama.ConsumerGroupSession) error {
	return nil
}

func (h *Handler[T]) Cleanup(session sarama.ConsumerGroupSession) error {
	return nil
}

func (h *Handler[T]) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	return h.consumeClaim(session, claim, time.Minute*time.Duration(h.cfg.ConsumeTime), true)
}

func (h *Handler[T]) consumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim, batchTimeout time.Duration, resetOnMessage bool) error {
	var events []T
	var msgRecords []*sarama.ConsumerMessage // 记录 Kafka 中还未消费的消息

	// 普通 Handler 使用空闲超时；BatchHandler 保持一秒批次窗口，不因新消息延长。
	timeout := time.NewTimer(batchTimeout)
	h.StopTimer(timeout)
	defer timeout.Stop()

	for {
		select {
		case <-session.Context().Done():
			// 会话退出时不再刷新缓冲区，未确认消息由下一次会话重投。
			return nil
		case msg, ok := <-claim.Messages():
			if !ok {
				return h.ConsumeEvents(&events, &msgRecords, session)
			}
			if session.Context().Err() != nil {
				return nil
			}
			var t T
			if err := json.Unmarshal(msg.Value, &t); err != nil {
				h.l.Error(LogKeyPartitionBlocked+" 反序列化消息体失败，保留位点并停止当前分区消费",
					logger.String("topic", msg.Topic),
					logger.Int32("partition", msg.Partition),
					logger.Int64("offset", msg.Offset),
					logger.Error(err))
				// 只允许确认坏消息之前成功处理的连续前缀，不能消费后续消息越过它。
				if consumeErr := h.ConsumeEvents(&events, &msgRecords, session); consumeErr != nil {
					return consumeErr
				}
				return fmt.Errorf("decode Kafka message %s/%d/%d: %w", msg.Topic, msg.Partition, msg.Offset, err)
			}

			events = append(events, t)
			msgRecords = append(msgRecords, msg)
			if len(events) >= h.cfg.ConsumeNum {
				h.StopTimer(timeout)
				if err := h.ConsumeEvents(&events, &msgRecords, session); err != nil {
					return err
				}
			} else if len(events) == 1 || resetOnMessage {
				h.StopTimer(timeout)
				timeout.Reset(batchTimeout)
			}
		case <-timeout.C:
			if err := h.ConsumeEvents(&events, &msgRecords, session); err != nil {
				return err
			}
		}
	}
}

func (h *Handler[T]) StopTimer(timeout *time.Timer) {
	if !timeout.Stop() {
		select {
		case <-timeout.C:
		default:
		}
	}
}

func (h *Handler[T]) ConsumeEvents(events *[]T, msgRecords *[]*sarama.ConsumerMessage, session sarama.ConsumerGroupSession) error {
	if len(*events) == 0 {
		return nil
	}
	first := (*msgRecords)[0]
	logh := h.l.With(logger.String("topic", first.Topic),
		logger.Int32("partition", first.Partition), logger.Int64("offset", first.Offset),
		logger.Int("batch_size", len(*events)))
	err := Retry(session.Context(), logh, func() error {
		return h.fn(session.Context(), *msgRecords, *events)
	})
	if err != nil {
		logh.Error(LogKeyPartitionBlocked+" 批量消费失败，保留位点并停止当前分区消费", logger.Error(err))
		return err
	}
	if err := session.Context().Err(); err != nil {
		return err
	}

	// 整个批次成功后才能确认位点；部分成功时整个批次可能重投，业务需保证幂等。
	for _, msg := range *msgRecords {
		session.MarkMessage(msg, "")
	}
	*events = nil
	*msgRecords = nil
	return nil
}
