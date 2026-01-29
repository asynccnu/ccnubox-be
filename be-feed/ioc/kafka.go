package ioc

import (
	"github.com/IBM/sarama"
	"github.com/asynccnu/ccnubox-be/be-feed/conf"
	"github.com/asynccnu/ccnubox-be/be-feed/events"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/infra"
	"github.com/asynccnu/ccnubox-be/common/pkg/saramax"
)

func InitKafka(cfg *conf.InfraConf) sarama.Client {
	return infra.InitKafka(cfg.Kafka)
}

func InitConsumers(
	feedEventConsumer *events.FeedEventConsumerHandler,
) []saramax.Consumer {
	return []saramax.Consumer{
		feedEventConsumer,
	}
}
