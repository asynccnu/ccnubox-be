package saramax

import (
	"context"
	"time"

	"github.com/IBM/sarama"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

type BatchHandler[T any] struct {
	l   logger.Logger
	cfg *HandlerConfig
	sk  *Skipper
	fn  func(msgs []*sarama.ConsumerMessage, t []T) error
}

func NewBatchHandler[T any](
	l logger.Logger,
	cfg *HandlerConfig,
	fn func(msgs []*sarama.ConsumerMessage, t []T) error) *BatchHandler[T] {
	return &BatchHandler[T]{
		l:   l,
		cfg: cfg,
		// 跳过计数要跨会话保持，否则每次 ConsumeClaim 重建后阈值永远累计不到。
		sk: NewSkipper(cfg.SkipAttempts, l),
		fn: fn,
	}
}

func (h *BatchHandler[T]) Setup(session sarama.ConsumerGroupSession) error {
	return nil
}

func (h *BatchHandler[T]) Cleanup(session sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim 复用统一的重试和位点确认逻辑，消息与反序列化结果保持一一对应。
func (h *BatchHandler[T]) ConsumeClaim(session sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim) error {
	handler := &Handler[T]{
		l:   h.l,
		cfg: h.cfg,
		sk:  h.sk,
		fn: func(_ context.Context, msgs []*sarama.ConsumerMessage, events []T) error {
			return h.fn(msgs, events)
		},
	}
	// claim 跑在 sarama 自己的协程里，panic 会直接崩掉进程，这里兜底成一次普通失败。
	return CatchPanic(h.l, func() error {
		return handler.consumeClaim(session, claim, time.Second, false)
	})
}
