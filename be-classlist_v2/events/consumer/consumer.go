package consumer

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/pkg/metricsx"
	"github.com/asynccnu/ccnubox-be/common/pkg/otelx/otelsarama"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// DelaySendHandler 消费延迟 topic消息并转发到真实 topic
type DelaySendHandler struct {
	topic         string
	kp            sarama.SyncProducer
	delayTime     time.Duration
	log           logger.Logger
	setOnce       sync.Once
	downOnce      sync.Once
	consumedTotal *prometheus.CounterVec
	mqFailedTotal *prometheus.CounterVec
}

func NewDelaySendHandler(topic string, client sarama.Client, delayTime time.Duration, l logger.Logger, m *metricsx.Metrics) (*DelaySendHandler, error) {
	kp, err := sarama.NewSyncProducerFromClient(client)
	if err != nil {
		return nil, err
	}
	return &DelaySendHandler{
		topic:         topic,
		kp:            kp,
		delayTime:     delayTime,
		log:           l,
		consumedTotal: m.MQ().ConsumedTotal,
		mqFailedTotal: m.MQ().FailedTotal,
	}, nil
}

func (c *DelaySendHandler) Setup(sarama.ConsumerGroupSession) error {
	c.setOnce.Do(func() {
		c.log.Infof("delay send handler setup")
	})
	return nil
}

func (c *DelaySendHandler) Cleanup(sarama.ConsumerGroupSession) error {
	c.downOnce.Do(func() {
		c.log.Infof("delay send handler cleanup")
	})
	return nil
}

func (c *DelaySendHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		ctx := otel.GetTextMapPropagator().Extract(context.Background(), otelsarama.NewConsumerMessageCarrier(message))

		tracer := otel.Tracer("delay-queue-consume")
		ctx, span := tracer.Start(ctx, "delay-queue-consume",
			trace.WithSpanKind(trace.SpanKindConsumer),
		)

		tlog := c.log.WithContext(ctx)
		dur := time.Since(message.Timestamp)

		tlog.Debugf("Message claimed: key:%s, value:%s, time_sub:%v", string(message.Key), string(message.Value), dur)

		if dur >= c.delayTime {
			if c.delayTime > 0 && dur >= 20*c.delayTime {
				session.MarkMessage(message, "")
				span.End()
				continue
			}

			err := c.forwardMessage(ctx, message)
			if err != nil {
				tlog.Errorf("Error forwarding message: %s", string(message.Value))
				if c.mqFailedTotal != nil {
					c.mqFailedTotal.WithLabelValues(c.topic, classifyError(err)).Inc()
				}
				span.End()
				return nil
			}

			// 消费计数
			if c.consumedTotal != nil {
				c.consumedTotal.WithLabelValues(c.topic, "OK").Inc()
			}

			session.MarkMessage(message, "")
			span.End()
			continue
		}

		span.End()
		time.Sleep(time.Second)
		return nil
	}
	return nil
}

func (c *DelaySendHandler) forwardMessage(ctx context.Context, msg *sarama.ConsumerMessage) error {
	otel.GetTextMapPropagator().Inject(ctx, otelsarama.NewConsumerMessageCarrier(msg))

	tlog := c.log.WithContext(ctx)

	_, _, err := c.kp.SendMessage(&sarama.ProducerMessage{
		Topic: c.topic,
		Key:   sarama.ByteEncoder(msg.Key),
		Value: sarama.ByteEncoder(msg.Value),
	})
	if err == nil {
		tlog.Debugf("Forwarded message: key=%s,val=%s,timestamp=%v, current-time=%v", string(msg.Key), string(msg.Value), msg.Timestamp, time.Now())
	}
	return err
}

// FuncConsumeHandler 消费真实 topic 消息并交付给应用
type FuncConsumeHandler struct {
	f             func(ctx context.Context, key []byte, value []byte)
	log           logger.Logger
	consumedTotal *prometheus.CounterVec
	mqFailedTotal *prometheus.CounterVec
}

func NewFuncConsumeHandler(f func(ctx context.Context, key []byte, value []byte), l logger.Logger, m *metricsx.Metrics) FuncConsumeHandler {
	return FuncConsumeHandler{
		f:             f,
		log:           l,
		consumedTotal: m.MQ().ConsumedTotal,
		mqFailedTotal: m.MQ().FailedTotal,
	}
}

func (fc FuncConsumeHandler) Setup(sarama.ConsumerGroupSession) error {
	fc.log.Info("Setting up func consume handler")
	return nil
}

func (fc FuncConsumeHandler) Cleanup(sarama.ConsumerGroupSession) error {
	fc.log.Info("Cleaning up func consume handler")
	return nil
}

func (fc FuncConsumeHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		ctx := otel.GetTextMapPropagator().Extract(context.Background(), otelsarama.NewConsumerMessageCarrier(message))

		tracer := otel.Tracer("real-topic")
		ctx, span := tracer.Start(ctx, "real_topic_consumer",
			trace.WithSpanKind(trace.SpanKindConsumer),
		)

		tlog := fc.log.WithContext(ctx)

		tlog.Debugf("Message claimed: key:%s, value:%s", string(message.Key), string(message.Value))
		fc.f(ctx, message.Key, message.Value)

		if fc.consumedTotal != nil {
			fc.consumedTotal.WithLabelValues(message.Topic, "OK").Inc()
		}
		session.MarkMessage(message, "")

		span.End()
	}
	return nil
}

type Consumer struct {
	cctx       context.Context
	cancelFunc context.CancelFunc
	client     sarama.Client
	log        logger.Logger
}

func NewConsumer(client sarama.Client, l logger.Logger) *Consumer {
	cctx, cancel := context.WithCancel(context.Background())
	return &Consumer{
		cctx:       cctx,
		cancelFunc: cancel,
		client:     client,
		log:        l,
	}
}

func (c *Consumer) Consume(topics []string, groupID string, handler sarama.ConsumerGroupHandler) error {
	cg, err := sarama.NewConsumerGroupFromClient(groupID, c.client)
	if err != nil {
		return err
	}
	defer cg.Close()

	for {
		if err := cg.Consume(c.cctx, topics, handler); err != nil {
			return err
		}
		if c.cctx.Err() != nil {
			return c.cctx.Err()
		}
	}
}

func (c *Consumer) Close() {
	if c.cancelFunc != nil {
		c.log.Infof("Consumer is shutting down, cancelling context")
		c.cancelFunc()
	}
}

var ErrInvalidGroupID = errors.New("the groupID is not allowed")

// classifyError 将 Kafka/Sarama 错误分类，用于 mq_failed_total 标签
func classifyError(err error) string {
	errStr := err.Error()
	if strings.Contains(errStr, "leader not available") {
		return "leader_not_available"
	}
	if strings.Contains(errStr, "not enough replicas") {
		return "not_enough_replicas"
	}
	if strings.Contains(errStr, "message too large") {
		return "message_too_large"
	}
	if strings.Contains(errStr, "invalid topic") {
		return "invalid_topic"
	}
	return "consume_error"
}
