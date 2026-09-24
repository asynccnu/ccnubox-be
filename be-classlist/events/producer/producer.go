package producer

import (
	"context"
	"errors"
	"time"

	"github.com/IBM/sarama"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/pkg/metricsx"
	"github.com/asynccnu/ccnubox-be/common/pkg/saramax"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type Producer struct {
	topic         string
	kp            syncProducer
	sendGuard     *saramax.SendGuard
	log           logger.Logger
	producedTotal *prometheus.CounterVec
	mqFailedTotal *prometheus.CounterVec
}

type syncProducer interface {
	SendMessage(msg *sarama.ProducerMessage) (partition int32, offset int64, err error)
	Close() error
}

func NewProducer(topic string, client sarama.Client, l logger.Logger, m *metricsx.Metrics, sendGuard *saramax.SendGuard) (*Producer, error) {
	kp, err := sarama.NewSyncProducerFromClient(client)
	if err != nil {
		return nil, err
	}
	return &Producer{
		topic:         topic,
		kp:            kp,
		sendGuard:     sendGuard,
		log:           l,
		producedTotal: m.MQMetrics.ProducedTotal,
		mqFailedTotal: m.MQMetrics.FailedTotal,
	}, nil
}

func (p *Producer) SendMessage(ctx context.Context, key, value []byte) error {
	tracer := otel.Tracer("delay-producer")
	ctx, span := tracer.Start(ctx, "delay_produce_message",
		trace.WithSpanKind(trace.SpanKindProducer),
	)
	defer span.End()

	tlog := p.log.WithContext(ctx)

	msg := &sarama.ProducerMessage{
		Topic:     p.topic,
		Key:       sarama.ByteEncoder(key),
		Value:     sarama.ByteEncoder(value),
		Timestamp: time.Now(),
	}

	// 包装使用ctx
	if err := ctx.Err(); err != nil {
		p.recordFailure(err)
		return err
	}

	err := p.sendGuard.Send(ctx, func() error {
		_, _, err := p.kp.SendMessage(msg)
		return err
	})
	if err != nil {
		span.RecordError(err)
		p.recordFailure(err)
		return err
	}
	if p.producedTotal != nil {
		p.producedTotal.WithLabelValues(p.topic, "OK").Inc()
	}
	tlog.Debugf("Produced message with key:%s, value:%s", string(key), string(value))
	return nil
}

func (p *Producer) recordFailure(err error) {
	if p.mqFailedTotal != nil {
		p.mqFailedTotal.WithLabelValues(p.topic, classifyError(err)).Inc()
	}
}

func (p *Producer) Close() {
	if err := p.sendGuard.Close(p.kp.Close); err != nil {
		p.log.Errorf("Error closing kp: %v", err)
		return
	}
	p.log.Infof("Producer closed successfully")
}

// classifyError 将 Kafka/Sarama 错误分类，用于 mq_failed_total 标签
func classifyError(err error) string {
	var producerErr *sarama.ProducerError
	if errors.As(err, &producerErr) {
		err = producerErr.Err
	}

	if errors.Is(err, sarama.ErrLeaderNotAvailable) {
		return "leader_not_available"
	}
	if errors.Is(err, sarama.ErrNotEnoughReplicas) || errors.Is(err, sarama.ErrNotEnoughReplicasAfterAppend) {
		return "not_enough_replicas"
	}
	if errors.Is(err, sarama.ErrMessageTooLarge) || errors.Is(err, sarama.ErrMessageSizeTooLarge) {
		return "message_too_large"
	}
	if errors.Is(err, sarama.ErrInvalidTopic) {
		return "invalid_topic"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	if errors.Is(err, context.Canceled) {
		return "canceled"
	}
	return "produce_error"
}
