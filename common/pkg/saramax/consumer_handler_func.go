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
	sk  *Skipper
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
		sk:  NewSkipper(cfg.SkipAttempts, l),
	}
}

func (h *Handler[T]) Setup(session sarama.ConsumerGroupSession) error {
	return nil
}

func (h *Handler[T]) Cleanup(session sarama.ConsumerGroupSession) error {
	return nil
}

func (h *Handler[T]) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	// claim 跑在 sarama 自己的协程里，panic 会直接崩掉进程，这里兜底成一次普通失败。
	return CatchPanic(h.l, func() error {
		return h.consumeClaim(session, claim, time.Minute*time.Duration(h.cfg.ConsumeTime), true)
	})
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
				// 只允许确认坏消息之前成功处理的连续前缀，不能消费后续消息越过它。
				if consumeErr := h.ConsumeEvents(&events, &msgRecords, session); consumeErr != nil {
					return consumeErr
				}
				// 反序列化失败是确定性的永久失败，按阈值决定保留位点还是跳过。
				if consumeErr := h.dropDecodeFailure(session, msg, err); consumeErr != nil {
					return consumeErr
				}
				continue
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
	logh := h.l.With(MessageFields(first)...).With(logger.Int("batch_size", len(*events)))
	err := Retry(session.Context(), logh, h.cfg.RetryAttempts, func() error {
		return CatchPanic(h.l, func() error {
			return h.fn(session.Context(), *msgRecords, *events)
		}, MessageFields(first)...)
	})
	if err != nil {
		if session.Context().Err() != nil {
			// 停机/再平衡导致的失败不是坏消息，位点会随下次会话重投，
			// 不能打 KAFKA_PARTITION_BLOCKED 造成误告警。
			logh.Info("会话结束，批次未确认，等待下次会话重投", logger.Error(err))
			return err
		}
		// 整批错误里含永久失败（消息体非法、字段非法等）时逐条定位：
		// 只跳过确认处理不了的消息，正常的消息照常确认。
		// 依赖故障（普通错误）直接保留整批位点——逐条重试只会放大故障。
		if IsPermanent(err) {
			return h.consumeOneByOne(events, msgRecords, session, logh)
		}
		logh.Error(LogKeyPartitionBlocked+" 批量消费失败，保留位点并停止当前分区消费", logger.Error(err))
		return err
	}
	if err := session.Context().Err(); err != nil {
		return err
	}

	// 整个批次成功后才能确认位点；部分成功时整个批次可能重投，业务需保证幂等。
	for _, msg := range *msgRecords {
		session.MarkMessage(msg, "")
		h.sk.Forget(msg)
	}
	*events = nil
	*msgRecords = nil
	return nil
}

// dropDecodeFailure 处理无法反序列化的消息：阈值内保留位点、停止分区消费等人工介入，
// 达到阈值后跳过它，避免分区长期停摆。
// 返回 nil 表示消息已跳过，可以继续消费后续消息。
func (h *Handler[T]) dropDecodeFailure(session sarama.ConsumerGroupSession, msg *sarama.ConsumerMessage, cause error) error {
	fields := MessageFields(msg)
	logh := h.l.With(fields...)
	count, reached := h.sk.Fail(msg)
	if !reached {
		logh.Error(LogKeyPartitionBlocked+" 反序列化消息体失败，保留位点并停止当前分区消费",
			logger.Int("attempts", count), logger.Error(cause))
		return fmt.Errorf("decode Kafka message %s/%d/%d: %w", msg.Topic, msg.Partition, msg.Offset, cause)
	}
	if err := session.Context().Err(); err != nil {
		// 停机/再平衡时确认无效，交给下次会话重投。
		return err
	}
	h.sk.Forget(msg)
	logh.Error(LogKeyMessageDropped+" 反序列化失败次数达到阈值，跳过该消息",
		logger.Int("attempts", count), logger.Error(cause))
	session.MarkMessage(msg, "")
	return nil
}

// consumeOneByOne 在整批失败后逐条重试，定位到底哪几条消息处理不了：
// 成功的消息照常确认（不再随整批重投），永久失败的消息按阈值决定跳过还是继续阻塞。
// 先收齐每条结果再统一确认，避免中途确认后面的消息越过前面的失败位点。
func (h *Handler[T]) consumeOneByOne(events *[]T, msgRecords *[]*sarama.ConsumerMessage, session sarama.ConsumerGroupSession, logh logger.Logger) error {
	msgs, evs := *msgRecords, *events
	results := make([]error, len(msgs))
	for i, msg := range msgs {
		if err := session.Context().Err(); err != nil {
			return err
		}
		err := CatchPanic(h.l, func() error {
			return h.fn(session.Context(), msgs[i:i+1], evs[i:i+1])
		}, MessageFields(msg)...)
		results[i] = err
	}

	// 是否可跳过由错误分类决定，不能因整批失败而绕过永久失败计数。
	// 只推进连续前缀，遇到未达阈值或临时失败的队首就停止确认。

	var blockingErr error
	for i, msg := range msgs {
		if err := session.Context().Err(); err != nil {
			return err
		}
		msgLog := logh.With(MessageFields(msg)...)
		err := results[i]
		if err == nil {
			h.sk.Forget(msg)
			session.MarkMessage(msg, "")
			continue
		}
		if !IsPermanent(err) {
			msgLog.Error(LogKeyPartitionBlocked+" 批量消费失败，保留位点并停止当前分区消费", logger.Error(err))
			blockingErr = err
			break
		}
		count, reached := h.sk.Fail(msg)
		if !reached {
			msgLog.Error(LogKeyPartitionBlocked+" 消息永久失败，保留位点等待人工处理",
				logger.Int("attempts", count), logger.Error(err))
			blockingErr = err
			break
		}
		h.sk.Forget(msg)
		msgLog.Error(LogKeyMessageDropped+" 消息永久失败次数达到阈值，跳过该消息",
			logger.Int("attempts", count), logger.Error(err))
		session.MarkMessage(msg, "")
	}
	*events = nil
	*msgRecords = nil
	return blockingErr
}

// MessageFields 返回日志里统一使用的消息位点字段。
func MessageFields(msg *sarama.ConsumerMessage) []logger.Field {
	return []logger.Field{
		logger.String("topic", msg.Topic),
		logger.Int32("partition", msg.Partition),
		logger.Int64("offset", msg.Offset),
	}
}
