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
	fn  func(msgs []*sarama.ConsumerMessage, t []T) error
}

func NewBatchHandler[T any](
	l logger.Logger,
	cfg *HandlerConfig,
	fn func(msgs []*sarama.ConsumerMessage, t []T) error) *BatchHandler[T] {
	return &BatchHandler[T]{
		l:   l,
		cfg: cfg,
		fn:  fn,
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
		fn: func(_ context.Context, msgs []*sarama.ConsumerMessage, events []T) error {
			return h.fn(msgs, events)
		},
	}
	return handler.consumeClaim(session, claim, time.Second, false)
}
