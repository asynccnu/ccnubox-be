package ioc

import (
	"time"

	"github.com/IBM/sarama"
	"github.com/asynccnu/ccnubox-be/be-feed/conf"
	"github.com/asynccnu/ccnubox-be/be-feed/events"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/infra"
	"github.com/asynccnu/ccnubox-be/common/pkg/saramax"
)

func InitKafka(cfg *conf.InfraConf) sarama.Client {
	return infra.InitKafka(cfg.Kafka, func(config *sarama.Config) {
		config.Producer.RequiredAcks = sarama.WaitForAll
		config.Producer.Timeout = 5 * time.Second
	})
}

func InitConsumers(
	feedEventConsumer *events.FeedEventConsumerHandler,
) []saramax.Consumer {
	return []saramax.Consumer{
		feedEventConsumer,
	}
}
