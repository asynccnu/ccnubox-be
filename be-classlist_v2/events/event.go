package events

import (
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/events/delay"
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	delay.NewDelayKafka,
	delay.NewDelayKafkaConfig,
)
