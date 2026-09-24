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

// Close 只关闭消费者组本身。NewConsumerGroupFromClient 不会关闭传入的 client，
// 停机时需由调用方先关闭 producer，再单独关闭共享的 sarama.Client。
func (c *saramaConsumer) Close() error {
	return c.consumerGroup.Close()
}
