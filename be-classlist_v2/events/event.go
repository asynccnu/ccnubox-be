package events

import (
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/events/consumer"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/events/delay"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/events/producer"
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	consumer.NewConsumer,
	consumer.NewDelaySendHandler,
	consumer.NewFuncConsumeHandler,
	delay.NewDelayKafka,
	delay.NewDelayKafkaConfig,
	producer.NewProducer,
)
