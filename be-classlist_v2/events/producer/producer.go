package producer

import (
	"context"
	"time"

	"github.com/IBM/sarama"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type Producer struct {
	topic string
	kp    sarama.SyncProducer
}

func NewProducer(topic string, client sarama.Client) (*Producer, error) {
	kp, err := sarama.NewSyncProducerFromClient(client)
	if err != nil {
		return nil, err
	}
	return &Producer{
		topic: topic,
		kp:    kp,
	}, nil
}

func (p *Producer) SendMessage(ctx context.Context, key, value []byte) error {
	tracer := otel.Tracer("delay-producer")
	ctx, span := tracer.Start(ctx, "delay_produce_message",
		trace.WithSpanKind(trace.SpanKindProducer),
	)
	defer span.End()

	tlog := logger.From(ctx)

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
	tlog.Debugf("Produced message with key:%s, value:%s", string(key), string(value))
	return nil
}

func (p *Producer) Close() {
	if err := p.kp.Close(); err != nil {
		logger.GlobalLogger.Errorf("Error closing kp: %v", err)
		return
	}
	logger.GlobalLogger.Infof("Producer closed successfully")
}
