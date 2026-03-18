package consumer

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/pkg/otelx/otelsarama"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// 因为 DelaySendHandler 实现的是 ConsumerGroupSession 本质上还是 comsumer 只不过他的消费逻辑是发送
type DelaySendHandler struct {
	topic     string
	kp        sarama.SyncProducer
	delayTime time.Duration
	sync.Once
}

func NewDelaySendHandler(topic string, client sarama.Client, delayTime time.Duration) (*DelaySendHandler, error) {
	kp, err := sarama.NewSyncProducerFromClient(client)
	if err != nil {
		return nil, err
	}
	return &DelaySendHandler{
		topic:     topic,
		kp:        kp,
		delayTime: delayTime,
	}, nil
}

func (c *DelaySendHandler) Setup(sarama.ConsumerGroupSession) error {
	c.Do(func() {
		logger.GlobalLogger.Infof("delay send handler setup")
	})
	return nil
}

func (c *DelaySendHandler) Cleanup(sarama.ConsumerGroupSession) error {
	c.Do(func() {
		logger.GlobalLogger.Infof("delay send handler cleanup")
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

		tlog := logger.From(ctx)
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
				span.End()
				return nil
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

	tlog := logger.From(ctx)

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

type FuncConsumeHandler struct {
	f func(ctx context.Context, key []byte, value []byte)
}

func NewFuncConsumeHandler(f func(ctx context.Context, key []byte, value []byte)) FuncConsumeHandler {
	return FuncConsumeHandler{
		f: f,
	}
}

func (fc FuncConsumeHandler) Setup(sarama.ConsumerGroupSession) error {
	logger.GlobalLogger.Info("Setting up func consume handler")
	return nil
}

func (fc FuncConsumeHandler) Cleanup(sarama.ConsumerGroupSession) error {
	logger.GlobalLogger.Info("Cleaning up func consume handler")
	return nil
}

func (fc FuncConsumeHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		ctx := otel.GetTextMapPropagator().Extract(context.Background(), otelsarama.NewConsumerMessageCarrier(message))

		tracer := otel.Tracer("real-topic")
		ctx, span := tracer.Start(ctx, "real_topic_consumer",
			trace.WithSpanKind(trace.SpanKindConsumer),
		)

		tlog := logger.From(ctx)

		tlog.Debugf("Message claimed: key:%s, value:%s", string(message.Key), string(message.Value))
		fc.f(ctx, message.Key, message.Value)
		session.MarkMessage(message, "")

		span.End()
	}
	return nil
}

type Consumer struct {
	cctx       context.Context
	cancelFunc context.CancelFunc
	client     sarama.Client
}

func NewConsumer(client sarama.Client) *Consumer {
	cctx, cancel := context.WithCancel(context.Background())
	return &Consumer{
		cctx:       cctx,
		cancelFunc: cancel,
		client:     client,
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
		logger.GlobalLogger.Infof("Consumer is shutting down, cancelling context")
		c.cancelFunc()
	}
}

var ErrInvalidGroupID = errors.New("the groupID is not allowed")
