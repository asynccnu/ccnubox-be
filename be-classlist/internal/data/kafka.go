package data

import (
	"context"
	"errors"
	"time"

	"github.com/IBM/sarama"
	"github.com/go-kratos/kratos/v2/log"
)

//使用kafka实现一个延迟消息队列

type producer struct {
	topic string
	kp    sarama.SyncProducer
	log   *log.Helper
}

func newProducer(topic string, kpb *KafkaProducerBuilder, logger *log.Helper) (*producer, error) {
	kp, err := kpb.Build()
	if err != nil {
		return nil, err
	}
	return &producer{
		topic: topic,
		kp:    kp,
		log:   logger,
	}, nil
}

func (p *producer) SendMessage(key, value []byte) error {
	msg := &sarama.ProducerMessage{
		Topic:     p.topic,
		Key:       sarama.ByteEncoder(key),
		Value:     sarama.ByteEncoder(value),
		Timestamp: time.Now(),
	}

	_, _, err := p.kp.SendMessage(msg)
	if err != nil {
		return err
	}
	p.log.Debugf("Produced message with key:%s, value:%s", string(key), string(value))
	return nil
}

func (p *producer) Close() {
	if err := p.kp.Close(); err != nil {
		p.log.Errorf("Error closing kp: %v", err)
		return
	}
	p.log.Infof("Producer closed successfully")
}

type delaySendHandler struct {
	topic     string
	kp        sarama.SyncProducer
	delayTime time.Duration
	log       *log.Helper
}

func newDelaySendHandler(topic string, kpb *KafkaProducerBuilder, delayTime time.Duration, logger *log.Helper) (*delaySendHandler, error) {
	kp, err := kpb.Build()
	if err != nil {
		return nil, err
	}
	return &delaySendHandler{
		topic:     topic,
		kp:        kp,
		delayTime: delayTime,
		log:       logger,
	}, nil
}

func (c *delaySendHandler) Setup(sarama.ConsumerGroupSession) error {
	c.log.Infof("delay send handler setup")
	return nil
}

func (c *delaySendHandler) Cleanup(sarama.ConsumerGroupSession) error {
	c.log.Infof("delay send handler cleanup")
	return nil
}

func (c *delaySendHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		dur := time.Now().Sub(message.Timestamp)
		c.log.Debugf("Message claimed: key:%s, value:%s,time_sub:%v", string(message.Key), string(message.Value), dur)
		// 当前是否超过延迟时间
		if dur >= c.delayTime {
			// 如果当前时间已经超过20倍的延迟时间，不转发消息，提交偏移量
			if c.delayTime > 0 && dur >= 20*c.delayTime {
				session.MarkMessage(message, "")
				continue
			}

			err := c.forwardMessage(message)
			if err != nil {
				c.log.Errorf("Error forwarding message: %s", string(message.Value))
				return nil
			}
			session.MarkMessage(message, "")
			continue
		}
		// 如果当前时间没有超过延迟时间,睡觉1秒,return,重新轮询
		time.Sleep(time.Second)
		return nil
	}
	return nil
}

// 转发消息到真实 topic
func (c *delaySendHandler) forwardMessage(msg *sarama.ConsumerMessage) error {
	_, _, err := c.kp.SendMessage(&sarama.ProducerMessage{
		Topic: c.topic,
		Key:   sarama.ByteEncoder(msg.Key),
		Value: sarama.ByteEncoder(msg.Value),
	})
	if err == nil {
		c.log.Debugf("Forwarded message: key=%s,val=%s,timestamp=%v, current-time=%v", string(msg.Key), string(msg.Value), msg.Timestamp, time.Now())
	}
	return err
}

// funcConsumeHandler 是一个函数类型，用于处理 Kafka 消息
type funcConsumeHandler struct {
	f   func(key []byte, value []byte)
	log *log.Helper
}

func newFuncConsumeHandler(log *log.Helper, f func(key []byte, value []byte)) funcConsumeHandler {
	return funcConsumeHandler{
		f:   f,
		log: log,
	}
}
func (fc funcConsumeHandler) Setup(sarama.ConsumerGroupSession) error {
	fc.log.Info("Setting up func consume handler")
	return nil
}
func (fc funcConsumeHandler) Cleanup(sarama.ConsumerGroupSession) error {
	fc.log.Info("Cleaning up func consume handler")
	return nil
}
func (fc funcConsumeHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		fc.log.Debugf("Message claimed: key:%s, value:%s", string(message.Key), string(message.Value))
		fc.f(message.Key, message.Value)
		session.MarkMessage(message, "")
	}
	return nil
}

// consumer 消费者
type consumer struct {
	cctx       context.Context
	cancelFunc context.CancelFunc
	kcb        *KafkaConsumerBuilder
	log        *log.Helper
}

func newConsumer(kcb *KafkaConsumerBuilder, logger *log.Helper) *consumer {
	cctx, cancel := context.WithCancel(context.Background())
	c := &consumer{
		cctx:       cctx,
		cancelFunc: cancel,
		kcb:        kcb,
		log:        logger,
	}
	return c
}

func (c *consumer) Consume(topics []string, groupID string, handler sarama.ConsumerGroupHandler) error {
	cg, err := c.kcb.Build(groupID)
	if err != nil {
		return err
	}

	defer cg.Close()

	// 循环消费（自动 rebalance 时必须循环）
	for {
		if err := cg.Consume(c.cctx, topics, handler); err != nil {
			return err
		}
		if c.cctx.Err() != nil {
			return c.cctx.Err()
		}
	}
}

func (c *consumer) Close() {
	if c.cancelFunc != nil {
		c.log.Infof("Consumer is shutting down, cancelling context")
		c.cancelFunc()
	}
}

// DelayKafka 是一个 Kafka 延迟消息队列的实现
// 对外提供 Send 和 Consume 方法
// Send 方法用于发送消息到延迟队列
// Consume 方法用于消费真实队列的消息
// 所以表面上看,DelayKafka 就像是只向一个队列里发消息，以及从同一个队列里消费消息
type DelayKafka struct {
	p          *producer
	c          *consumer
	delaySend  *delaySendHandler
	delayTopic string
	realTopic  string
	delayTime  time.Duration

	proxyGroupID string
	log          *log.Helper
}

type DelayKafkaConfig struct {
	delayTopic string
	realTopic  string
	delayTime  time.Duration
}

func NewDelayKafkaConfig() DelayKafkaConfig {
	return DelayKafkaConfig{
		delayTopic: "be-classlist-delay",
		realTopic:  "be-classlist-real",
		delayTime:  5 * time.Minute,
	}
}

func NewDelayKafka(kpb *KafkaProducerBuilder, kcb *KafkaConsumerBuilder, cf DelayKafkaConfig, logger log.Logger) (*DelayKafka, func(), error) {
	dk := &DelayKafka{
		delayTopic:   cf.delayTopic,
		realTopic:    cf.realTopic,
		delayTime:    cf.delayTime,
		proxyGroupID: "be-classlist-delay",
		log:          log.NewHelper(logger),
	}
	p, err := newProducer(dk.delayTopic, kpb, dk.log)
	if err != nil {
		return nil, nil, err
	}
	ds, err := newDelaySendHandler(dk.realTopic, kpb, dk.delayTime, dk.log)
	if err != nil {
		return nil, nil, err
	}
	c := newConsumer(kcb, dk.log)

	dk.p = p
	dk.c = c
	dk.delaySend = ds

	go func() {
		// 监听延迟队列的消息，并转发到真实队列
		if err := dk.consumeDelay(); err != nil {
			dk.log.Errorf("Error consuming delay topic: %v", err)
		}
	}()

	return dk, dk.Close, nil
}

// Send 发送消息到延迟队列
func (d *DelayKafka) Send(key, value []byte) error {
	return d.p.SendMessage(key, value)
}

func (d *DelayKafka) consumeDelay() error {
	return d.c.Consume([]string{d.delayTopic}, d.proxyGroupID, d.delaySend)
}

// Consume 消费真实队列的消息
func (d *DelayKafka) Consume(groupID string, f func(key, value []byte)) error {
	if groupID == d.proxyGroupID {
		return errors.New("the groupID is not allowed")
	}

	// 使用 funcConsumeHandler 将函数转换为 sarama.ConsumerGroupHandler
	handler := newFuncConsumeHandler(d.log, f)
	return d.c.Consume([]string{d.realTopic}, groupID, handler)
}

func (d *DelayKafka) Close() {
	if d.p != nil {
		d.p.Close()
	}
	if d.c != nil {
		d.c.Close()
	}
}
