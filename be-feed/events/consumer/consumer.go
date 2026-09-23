package consumer

import (
	"context"
	"github.com/IBM/sarama"
)

// 定义一个空的接口
type Consumer interface {
	Consume(ctx context.Context, topics []string, handler sarama.ConsumerGroupHandler) error
	Close() error
}

// SaramaProducer 使用 sarama.Client 的生产者实现
type saramaConsumer struct {
	consumerGroup sarama.ConsumerGroup
}

func NewSaramaConsumer(kafkaClient sarama.Client, feedGroup string) Consumer {
	// 创建一个新的消费者组，组名为 "feed-event-sync"
	cg, err := sarama.NewConsumerGroupFromClient(feedGroup, kafkaClient)
	if err != nil {
		panic("创建消费者失败") // 如果创建消费者组失败，返回错误
	}

	return &saramaConsumer{consumerGroup: cg}
}

// 随便包了一层,主要是比较方便统一更改设定
func (c *saramaConsumer) Consume(ctx context.Context, topics []string, handler sarama.ConsumerGroupHandler) error {
	return c.consumerGroup.Consume(ctx, topics, handler)
}

// Close 会同时关闭创建消费者组时传入的 client（sarama 语义），
// 因此停机时它会一起影响共用该 client 的生产者，进程退出前不需要再单独关闭。
func (c *saramaConsumer) Close() error {
	return c.consumerGroup.Close()
}
