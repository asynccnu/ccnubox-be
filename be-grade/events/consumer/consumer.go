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

// Close 只结束消费组会话并退出消费者组。
// 由于这里用的是 NewConsumerGroupFromClient，sarama 会把传入的 client 包在 nopCloserClient 里，
// 关闭消费组并不会关闭共享的 client（当前进程退出前也无人显式关闭它，连接随进程回收）；
// 若后续需要确定性关闭，必须在生产者也停止之后单独关闭该 client。
func (c *saramaConsumer) Close() error {
	return c.consumerGroup.Close()
}
