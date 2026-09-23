package saramax

import (
	"context"
	"math/rand/v2"
	"time"
)

const maxRetryBackoff = 30 * time.Second

// Backoff 使用有上限的指数退避和抖动，避免故障恢复时集中请求 Kafka。
// 每个消费循环单独持有，不在多个协程间共享。
type Backoff struct {
	delay time.Duration
}

func (b *Backoff) Next() time.Duration {
	if b.delay == 0 {
		b.delay = time.Second
	} else {
		b.delay = min(b.delay*2, maxRetryBackoff)
	}
	if b.delay == maxRetryBackoff {
		return b.delay*4/5 + time.Duration(rand.Int64N(int64(b.delay/5)+1))
	}
	return b.delay + time.Duration(rand.Int64N(int64(b.delay/4)+1))
}

func (b *Backoff) Wait(ctx context.Context) error {
	return wait(ctx, b.Next())
}

func (b *Backoff) Reset() {
	b.delay = 0
}
