package saramax

import (
	"github.com/IBM/sarama"
)

type HandlerFunc func(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error

func (h HandlerFunc) Setup(session sarama.ConsumerGroupSession) error {
	return nil
}

func (h HandlerFunc) Cleanup(session sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim 兜底自定义消费循环的 panic：claim 跑在 sarama 自己的协程里，
// 这里不兜底会直接崩掉整个进程（没有 logger 时只转成错误，由调用方记录）。
func (h HandlerFunc) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	return CatchPanic(nil, func() error {
		return h(session, claim)
	})
}
