package saramax

import (
	"context"
	"log"
)

// TODO 待完善的pkg
type Consumer interface {
	Start() error
}

// stopper 由具体的消费者实现，用于停机时释放会话并确认已处理的位点。
type stopper interface {
	Stop()
}

// StopConsumers 有界地等待消费者停机：消费者可能正卡在一批消息上（单批可达上百条），
// 超时后不再阻塞进程退出，未确认的位点会在下次启动时重投。
// 返回 false 表示等待超时。
func StopConsumers(ctx context.Context, consumers []Consumer) bool {
	done := make(chan struct{})
	go func() {
		defer close(done)
		for _, c := range consumers {
			if s, ok := c.(stopper); ok {
				s.Stop()
			} else {
				// 新增消费者忘了实现 Stop 时停机会静默跳过，只能靠这里提醒。
				log.Printf("KAFKA consumer %T 未实现 Stop()，停机时不再等待其退出", c)
			}
		}
	}()
	select {
	case <-done:
		return true
	case <-ctx.Done():
		return false
	}
}

type HandlerConfig struct {
	ConsumeTime int `yaml:"consumeTime"`
	ConsumeNum  int `yaml:"consumeNum"`
	// RetryAttempts 是单次批次在会话内的重试次数上限，<=0 时按默认 4 次。
	// 失败批次最终仍会结束会话并靠 Kafka 重投，长周期故障主要交给消费循环的外层退避。
	RetryAttempts int `yaml:"retryAttempts"`
}
