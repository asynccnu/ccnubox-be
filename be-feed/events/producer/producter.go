package producer

import (
	"encoding/json"
	"errors"
	"log"

	"github.com/IBM/sarama"
	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"github.com/asynccnu/ccnubox-be/common/pkg/metricsx"
	"github.com/prometheus/client_golang/prometheus"
)

// Producer 接口定义了 Kafka Producer 的行为
type Producer interface {
	SendMessage(topic string, msgData domain.FeedEvent) error
	Close() error
}

// SaramaProducer 使用 sarama.Client 的生产者实现
type saramaProducer struct {
	producer sarama.SyncProducer
}

// NewSaramaProducer 创建一个新的 SaramaProducer 实例
func NewSaramaProducer(kafkaClient sarama.Client) Producer {
	// 使用 Kafka 客户端创建同步生产者
	producer, err := sarama.NewSyncProducerFromClient(kafkaClient)
	if err != nil {
		log.Println("Failed to create sync producer:", err)
		return nil
	}

	return &saramaProducer{producer: producer}
}

// SendMessage 发送一条消息到指定的 Kafka 主题
func (p *saramaProducer) SendMessage(topic string, msgData domain.FeedEvent) error {
	//序列化
	data, err := json.Marshal(msgData)
	if err != nil {
		return err
	}
	//存储数据
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(data),
	}

	_, _, err = p.producer.SendMessage(msg)
	if err != nil {
		return err
	}

	return nil
}

// Close 关闭 Kafka Client
func (p *saramaProducer) Close() error {
	return p.producer.Close()
}

// instrumentedProducer 包装 Producer 接口，添加 metrics
type instrumentedProducer struct {
	Producer
	producedTotal *prometheus.CounterVec
	mqFailedTotal *prometheus.CounterVec
}

// NewInstrumentedProducer 创建带 metrics 的 Producer 包装器
func NewInstrumentedProducer(p Producer, producedTotal *prometheus.CounterVec, mqFailedTotal *prometheus.CounterVec) Producer {
	return &instrumentedProducer{
		Producer:      p,
		producedTotal: producedTotal,
		mqFailedTotal: mqFailedTotal,
	}
}

func NewInstrumentedSaramaProducer(kafkaClient sarama.Client, m *metricsx.Metrics) Producer {
	return NewInstrumentedProducer(NewSaramaProducer(kafkaClient), m.MQMetrics.ProducedTotal, m.MQMetrics.FailedTotal)
}

func (p *instrumentedProducer) SendMessage(topic string, msgData domain.FeedEvent) error {
	err := p.Producer.SendMessage(topic, msgData)
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

// classifyError 将 Kafka/Sarama 错误分类
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
