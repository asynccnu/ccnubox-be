package producer

import (
	"encoding/json"
	"github.com/IBM/sarama"
	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"log"
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
