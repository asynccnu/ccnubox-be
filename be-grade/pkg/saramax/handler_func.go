package saramax

import (
	"github.com/IBM/sarama"
)

type HandlerFunc func(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error

func (h HandlerFunc) Setup(session sarama.ConsumerGroupSession) error {
	return nil
}

func (h HandlerFunc) Cleanup(session sarama.ConsumerGroupSession) error {
	return nil
}

func (h HandlerFunc) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	return h(session, claim)
}
