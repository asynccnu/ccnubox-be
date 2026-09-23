package producer

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/IBM/sarama"
	"github.com/asynccnu/ccnubox-be/be-grade/domain"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/pkg/metricsx"
	"github.com/asynccnu/ccnubox-be/common/pkg/saramax"
	"github.com/prometheus/client_golang/prometheus"
)

// Producer 接口定义了 Kafka Producer 的行为
type Producer interface {
	SendMessage(ctx context.Context, topic string, msgData domain.NeedDetailGrade) error
	Close() error
}

// SaramaProducer 使用 sarama.Client 的生产者实现
type saramaProducer struct {
	producer  sarama.SyncProducer
	sendGuard *saramax.SendGuard
}

// NewSaramaProducer 创建一个新的 SaramaProducer 实例
func NewSaramaProducer(kafkaClient sarama.Client, l logger.Logger) Producer {
	// 使用 Kafka 客户端创建同步生产者
	producer, err := sarama.NewSyncProducerFromClient(kafkaClient)
	if err != nil {
		log.Println("Failed to create sync producer:", err)
		return nil
	}

	// 令牌桶：稳态 50 次/秒、突发 100 次。详情事件在用户刷新的 30 秒超时内发送，
	// 突发额度保证正常刷新不会因为限速拿不到令牌。
	// logger 用于输出进入/退出冷却的关键字日志（KAFKA_SEND_FAILED），不能传 nil。
	return &saramaProducer{producer: producer, sendGuard: saramax.NewSendGuard(50, 100, l)}
}

// SendMessage 发送一条消息到指定的 Kafka 主题
func (p *saramaProducer) SendMessage(ctx context.Context, topic string, msgData domain.NeedDetailGrade) error {
	//序列化
	data, err := json.Marshal(msgData)
	if err != nil {
		return err
	}
	//存储数据
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(msgData.StudentID),
		Value: sarama.ByteEncoder(data),
	}

	return p.sendGuard.Send(ctx, func() error {
		_, _, err := p.producer.SendMessage(msg)
		return err
	})
}

// Close 关闭 Kafka Client
func (p *saramaProducer) Close() error {
	return p.sendGuard.Close(p.producer.Close)
}

// instrumentedProducer 包装 Producer 接口，添加 metrics
type instrumentedProducer struct {
	Producer
	producedTotal *prometheus.CounterVec
	mqFailedTotal *prometheus.CounterVec
}

// NewInstrumentedProducer 创建带 metrics 的 Producer包装器
func NewInstrumentedProducer(p Producer, producedTotal *prometheus.CounterVec, mqFailedTotal *prometheus.CounterVec) Producer {
	return &instrumentedProducer{
		Producer:      p,
		producedTotal: producedTotal,
		mqFailedTotal: mqFailedTotal,
	}
}

func NewInstrumentedSaramaProducer(kafkaClient sarama.Client, l logger.Logger, m *metricsx.Metrics) Producer {
	return NewInstrumentedProducer(NewSaramaProducer(kafkaClient, l), m.MQMetrics.ProducedTotal, m.MQMetrics.FailedTotal)
}

func (p *instrumentedProducer) SendMessage(ctx context.Context, topic string, msgData domain.NeedDetailGrade) error {
	err := p.Producer.SendMessage(ctx, topic, msgData)
	if err != nil {
		if p.mqFailedTotal != nil {
			p.mqFailedTotal.WithLabelValues(topic, classifyError(err)).Inc()
		}
		return err
	}
	if p.producedTotal != nil {
		p.producedTotal.WithLabelValues(topic, "OK").Inc()
	}
	return nil
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
	return "produce_error"
}
