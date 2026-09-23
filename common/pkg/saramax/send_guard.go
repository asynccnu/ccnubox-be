package saramax

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

var (
	ErrProducerCoolingDown = errors.New("Kafka producer is cooling down after a failed send")
	ErrProducerClosed      = errors.New("Kafka producer is closed")
)

// SendGuard 用令牌桶限制发送速率并允许突发额度：
// 空闲时不需要额外等待，等待中的发送不占用任何共享资源，
// 不会出现旧实现那种“一个发送等待间隔时把其他发送一起堵住”的队头阻塞，
// 因此单条交互式发送不会被批量任务长期挤占（等待时间只取决于令牌补充速度与并发方数量）。
// 发送失败时共享冷却期，避免大量业务请求各自重试放大 Kafka 故障。
// 不异步缓存消息；限流、取消或发送失败必须由调用方处理。
type SendGuard struct {
	l     logger.Logger
	rate  float64 // 每秒补充的令牌数
	burst float64 // 桶容量，即允许瞬时消耗的发送次数

	mu       sync.Mutex
	tokens   float64
	last     time.Time
	cooldown time.Time
	rejected int
	backoff  Backoff
	closed   bool
	inflight sync.WaitGroup
}

// NewSendGuard rate 为稳态发送速率（次/秒），burst 为突发额度（次）。
// 初始令牌是满的，因此短时间的正常流量不会因为限速产生额外延迟。
// l 可以为 nil；发送失败和被拒绝的发送会输出关键字日志，便于故障检索。
func NewSendGuard(rate float64, burst int, l logger.Logger) *SendGuard {
	if rate <= 0 {
		rate = 1
	}
	if burst < 1 {
		burst = 1
	}
	return &SendGuard{
		l:      l,
		rate:   rate,
		burst:  float64(burst),
		tokens: float64(burst),
		last:   time.Now(),
	}
}

// Send 取到令牌后执行同步发送。并发在途的发送数量由调用方自身的并发度决定，
// 令牌桶只约束平均速率；单个发送的网络超时仍由 Kafka 配置限制。
func (g *SendGuard) Send(ctx context.Context, fn func() error) error {
	if err := g.acquire(ctx); err != nil {
		return err
	}
	defer g.inflight.Done()

	// Sarama 的同步发送不支持 ctx；网络超时和内部重试另由配置限制。
	err := fn()
	now := time.Now()
	g.mu.Lock()
	if err != nil {
		g.cooldown = now.Add(g.backoff.Next())
		g.mu.Unlock()
		g.logSendFailed(err)
		return err
	}
	// 冷却不存在，或本次发送是冷却到期后的探测（成功时间已过冷却截止点）：恢复正常。
	// 若冷却仍在生效期，说明是冷却设置前就已出发的在途发送成功，
	// 不能据此清除冷却，否则并发下持续故障期的退避会被反复清零。
	if g.cooldown.IsZero() || now.After(g.cooldown) {
		g.cooldown = time.Time{}
		g.backoff.Reset()
		rejected := g.rejected
		g.rejected = 0
		g.mu.Unlock()
		if rejected > 0 {
			g.logCooldownRecovered(rejected)
		}
		return nil
	}
	g.mu.Unlock()
	return nil
}

// acquire 校验关闭与冷却状态，等待令牌并标记在途发送。
func (g *SendGuard) acquire(ctx context.Context) error {
	for {
		g.mu.Lock()
		if g.closed {
			g.mu.Unlock()
			return ErrProducerClosed
		}
		if time.Now().Before(g.cooldown) {
			// 冷却期内不向 Kafka 发起请求，只统计被拒绝的次数。
			g.rejected++
			g.mu.Unlock()
			return ErrProducerCoolingDown
		}
		now := time.Now()
		g.tokens = min(g.burst, g.tokens+now.Sub(g.last).Seconds()*g.rate)
		g.last = now
		if g.tokens >= 1 {
			g.tokens--
			g.inflight.Add(1)
			g.mu.Unlock()
			return nil
		}
		// 令牌不足时只等待补满一个令牌的时间，不占用在途额度，
		// 因此等待中的发送不会阻塞其他调用方。
		waitFor := max(time.Duration((1-g.tokens)/g.rate*float64(time.Second)), time.Millisecond)
		g.mu.Unlock()
		if err := wait(ctx, waitFor); err != nil {
			return err
		}
	}
}

// Close 等待在途发送结束，阻止取消后仍存活的业务任务继续向 Kafka 发布。
// 多个生产者共享预算时，仍需分别调用其关闭函数。
func (g *SendGuard) Close(fn func() error) error {
	g.mu.Lock()
	g.closed = true
	g.mu.Unlock()
	g.inflight.Wait()
	return fn()
}

func (g *SendGuard) logSendFailed(err error) {
	if g.l == nil {
		return
	}
	g.l.Error(LogKeySendFailed+" 发送 Kafka 失败，进入共享冷却期，期间不再向 Kafka 发起请求",
		logger.Error(err))
}

func (g *SendGuard) logCooldownRecovered(rejected int) {
	if g.l == nil {
		return
	}
	g.l.Warn(LogKeySendFailed+" 发送已恢复，冷却期内被拒绝的消息没有进入 Kafka，需要业务按幂等 key 补发",
		logger.Int("rejected", rejected))
}
