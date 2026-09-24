package ioc

import (
	"time"

	"github.com/IBM/sarama"
	"github.com/asynccnu/ccnubox-be/be-classlist/conf"
	"github.com/asynccnu/ccnubox-be/be-classlist/events/consumer"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/infra"
)

const (
	kafkaDialTimeout     = 3 * time.Second
	kafkaReadTimeout     = 10 * time.Second
	kafkaWriteTimeout    = 5 * time.Second
	kafkaProducerTimeout = 5 * time.Second
	kafkaProducerRetries = 1
)

func InitKafka(cfg *conf.InfraConf) sarama.Client {
	return infra.InitKafka(cfg.Kafka, configureKafkaProducer)
}

func InitKafkaConsumerClientFactory(cfg *conf.InfraConf) consumer.ClientFactory {
	return func() (sarama.Client, error) {
		return infra.NewKafka(cfg.Kafka)
	}
}

func configureKafkaProducer(cfg *sarama.Config) {
	cfg.Net.DialTimeout = kafkaDialTimeout
	cfg.Net.ReadTimeout = kafkaReadTimeout
	cfg.Net.WriteTimeout = kafkaWriteTimeout
	cfg.Producer.Timeout = kafkaProducerTimeout
	cfg.Producer.Retry.Max = kafkaProducerRetries
}
