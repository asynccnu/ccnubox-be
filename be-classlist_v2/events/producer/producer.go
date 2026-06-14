package producer

import (
	"context"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/pkg/metricsx"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type Producer struct {
	topic         string
	kp            sarama.SyncProducer
	log           logger.Logger
	producedTotal *prometheus.CounterVec
	mqFailedTotal *prometheus.CounterVec
}

func NewProducer(topic string, client sarama.Client, l logger.Logger, m *metricsx.Metrics) (*Producer, error) {
	kp, err := sarama.NewSyncProducerFromClient(client)
	if err != nil {
		return nil, err
	}
	return &Producer{
		topic:         topic,
		kp:            kp,
		log:           l,
		producedTotal: m.MQ().ProducedTotal,
		mqFailedTotal: m.MQ().FailedTotal,
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

	_, _, err := p.kp.SendMessage(msg)
	if err != nil {
		if p.mqFailedTotal != nil {
			p.mqFailedTotal.WithLabelValues(p.topic, classifyError(err)).Inc()
		}
		return err
	}
	if p.producedTotal != nil {
		p.producedTotal.WithLabelValues(p.topic, "OK").Inc()
	}
	tlog.Debugf("Produced message with key:%s, value:%s", string(key), string(value))
	return nil
}

func (p *Producer) Close() {
	if err := p.kp.Close(); err != nil {
		p.log.Errorf("Error closing kp: %v", err)
		return
	}
	p.log.Infof("Producer closed successfully")
}

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
	return "produce_error"
}
