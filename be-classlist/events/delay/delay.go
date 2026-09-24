package delay

import (
	"context"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/asynccnu/ccnubox-be/be-classlist/biz"
	"github.com/asynccnu/ccnubox-be/be-classlist/events/consumer"
	"github.com/asynccnu/ccnubox-be/be-classlist/events/producer"
	"github.com/asynccnu/ccnubox-be/be-classlist/events/topic"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/pkg/metricsx"
	"github.com/asynccnu/ccnubox-be/common/pkg/saramax"
)

type DelayKafka struct {
	client       sarama.Client
	p            *producer.Producer
	c            *consumer.Consumer
	delaySend    *consumer.DelaySendHandler
	log          logger.Logger
	delayTopic   string
	realTopic    string
	delayTime    time.Duration
	proxyGroupID string
	m            *metricsx.Metrics
	ctx          context.Context // 消费循环的生命周期，Close 时取消
	cancel       context.CancelFunc
	closeOnce    sync.Once
}

type DelayKafkaConfig struct {
	DelayTopic string
	RealTopic  string
	DelayTime  time.Duration
}

func NewDelayKafkaConfig() DelayKafkaConfig {
	return DelayKafkaConfig{
		DelayTopic: topic.DelayTopic,
		RealTopic:  topic.RealTopic,
		DelayTime:  5 * time.Minute,
	}
}

// NewDelayKafka 接管生产者共享 client，初始化失败或 Close 时在生产者之后释放。
func NewDelayKafka(client sarama.Client, newConsumerClient consumer.ClientFactory, cf DelayKafkaConfig, l logger.Logger, m *metricsx.Metrics) (biz.DelayQueue, func(), error) {
	ctx, cancel := context.WithCancel(context.Background())
	dk := &DelayKafka{
		client:       client,
		delayTopic:   cf.DelayTopic,
		realTopic:    cf.RealTopic,
		delayTime:    cf.DelayTime,
		proxyGroupID: topic.DelayTopic,
		log:          l,
		m:            m,
		ctx:          ctx,
		cancel:       cancel,
	}

	// 入延迟队列和到期转发共享发送预算：令牌桶稳态 50 次/秒、突发 100 次，
	// 令牌不足时按补充速度等待，实际等待还取决于并发发送量。
	// 只有临时失败的实际冷却窗口会拒绝其他发送，到期后的成功发送会重置退避。
	guard := saramax.NewSendGuard(50, 100, l)
	p, err := producer.NewProducer(dk.delayTopic, client, l, m, guard)
	if err != nil {
		cancel()
		_ = client.Close()
		return nil, nil, err
	}
	ds, err := consumer.NewDelaySendHandler(dk.delayTopic, dk.realTopic, client, dk.delayTime, l, m, guard)
	if err != nil {
		p.Close()
		cancel()
		_ = client.Close()
		return nil, nil, err
	}
	c := consumer.NewConsumer(newConsumerClient, l)

	dk.p = p
	dk.c = c
	dk.delaySend = ds

	go func() {
		if err := dk.consumeDelay(); err != nil {
			dk.log.Errorf("Error consuming delay topic: %v", err)
		}
	}()

	return dk, dk.Close, nil
}

func (d *DelayKafka) Send(ctx context.Context, key, value []byte) error {
	return d.p.SendMessage(ctx, key, value)
}

func (d *DelayKafka) consumeDelay() error {
	return d.c.Consume(d.ctx, []string{d.delayTopic}, d.proxyGroupID, d.delaySend)
}

func (d *DelayKafka) Consume(groupID string, f func(ctx context.Context, key []byte, value []byte) (ack bool, err error)) error {
	if groupID == d.proxyGroupID {
		return consumer.ErrInvalidGroupID
	}
	handler := consumer.NewFuncConsumeHandler(f, d.log, d.m)
	return d.c.Consume(d.ctx, []string{d.realTopic}, groupID, handler)
}

func (d *DelayKafka) Close() {
	d.closeOnce.Do(func() {
		// 先取消消费循环（等待它们退出释放位点），再关转发生产者。
		d.cancel()
		if d.c != nil {
			d.c.Close()
		}
		if d.delaySend != nil {
			if err := d.delaySend.Close(); err != nil {
				d.log.Error("关闭延迟转发生产者失败", logger.Error(err))
			}
		}
		if d.p != nil {
			d.p.Close()
		}
		if d.client != nil {
			if err := d.client.Close(); err != nil {
				d.log.Error("关闭 Kafka client 失败", logger.Error(err))
			}
		}
	})
}
