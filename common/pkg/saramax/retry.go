package saramax

import (
	"context"
	"errors"
	"time"

	"github.com/IBM/sarama"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

// Retry 在当前会话内有限退避重试，取消后不再发起新的处理。
// fn 可能被重复调用，业务处理必须保证幂等。
func Retry(ctx context.Context, l logger.Logger, fn func() error) error {
	const maxAttempts = 4
	for attempt := 1; ; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := fn()
		if err == nil || attempt == maxAttempts {
			return err
		}
		delay := time.Duration(1<<(attempt-1)) * 100 * time.Millisecond
		l.Warn(LogKeyConsumeRetry+" 消费失败，退避后重试", logger.Int("attempt", attempt),
			logger.Int64("retry_delay_ms", delay.Milliseconds()), logger.Error(err))
		if err := wait(ctx, delay); err != nil {
			return err
		}
	}
}

// RunConsumer 在会话结束后重新消费，失败消息仍由 Kafka 保留，不跳过位点。
func RunConsumer(ctx context.Context, cg interface {
	Consume(context.Context, []string, sarama.ConsumerGroupHandler) error
}, topics []string, handler sarama.ConsumerGroupHandler, l logger.Logger) error {
	var backoff Backoff
	for ctx.Err() == nil {
		started := time.Now()
		err := cg.Consume(ctx, topics, handler)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if errors.Is(err, sarama.ErrClosedConsumerGroup) {
			return err
		}
		if err != nil {
			l.Error(LogKeyConsumeRetry+" 消费循环异常，退避后重新消费", logger.Error(err))
		}
		// ConsumeClaim 失败时 Sarama 也可能返回 nil，不能据此清零退避。
		// 只有持续运行的正常会话才恢复初始间隔，避免坏消息造成 rejoin 风暴。
		if err == nil && time.Since(started) >= time.Minute {
			backoff.Reset()
		}
		if err := backoff.Wait(ctx); err != nil {
			return err
		}
	}
	return ctx.Err()
}

func wait(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return ctx.Err()
	}
}
