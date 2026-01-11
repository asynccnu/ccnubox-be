package ioc

import (
	"github.com/IBM/sarama"
	"github.com/asynccnu/ccnubox-be/be-feed/conf"
	"github.com/asynccnu/ccnubox-be/be-feed/events"
	"github.com/asynccnu/ccnubox-be/be-feed/pkg/saramax"
)

func InitKafka(cfg *conf.InfraConf) sarama.Client {
	saramaCfg := sarama.NewConfig()
	saramaCfg.Producer.Return.Successes = true
	saramaCfg.Producer.Partitioner = sarama.NewConsistentCRCHashPartitioner
	client, err := sarama.NewClient(cfg.Kafka.Addrs, saramaCfg)
	if err != nil {
		panic(err)
	}
	return client
}

func InitConsumers(
	feedEventConsumer *events.FeedEventConsumerHandler,
) []saramax.Consumer {
	return []saramax.Consumer{
		feedEventConsumer,
	}
}
