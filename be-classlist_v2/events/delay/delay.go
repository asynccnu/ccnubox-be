package delay

import (
	"context"
	"time"

	"github.com/IBM/sarama"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/biz"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/events/consumer"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/events/producer"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/events/topic"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/pkg/metricsx"
)

type DelayKafka struct {
	p            *producer.Producer
	c            *consumer.Consumer
	delaySend    *consumer.DelaySendHandler
	log          logger.Logger
	delayTopic   string
	realTopic    string
	delayTime    time.Duration
	proxyGroupID string
	m            *metricsx.Metrics
}

type DelayKafkaConfig struct {
	DelayTopic string
	RealTopic  string
	DelayTime  time.Duration
}

func NewDelayKafkaConfig() DelayKafkaConfig {
	return DelayKafkaConfig{
		DelayTopic: topic.DelayTopic,
		RealTopic:  topic.RealTopic,
		DelayTime:  5 * time.Minute,
	}
}

func NewDelayKafka(client sarama.Client, cf DelayKafkaConfig, l logger.Logger, m *metricsx.Metrics) (biz.DelayQueue, func(), error) {
	dk := &DelayKafka{
		delayTopic:   cf.DelayTopic,
		realTopic:    cf.RealTopic,
		delayTime:    cf.DelayTime,
		proxyGroupID: topic.DelayTopic,
		log:          l,
		m:            m,
	}

	p, err := producer.NewProducer(dk.delayTopic, client, l, m)
	if err != nil {
		return nil, nil, err
	}
	ds, err := consumer.NewDelaySendHandler(dk.realTopic, client, dk.delayTime, l, m)
	if err != nil {
		return nil, nil, err
	}
	c := consumer.NewConsumer(client, l)

	dk.p = p
	dk.c = c
	dk.delaySend = ds

	go func() {
		if err := dk.consumeDelay(); err != nil {
			dk.log.Errorf("Error consuming delay topic: %v", err)
		}
	}()

	return dk, dk.Close, nil
}

func (d *DelayKafka) Send(ctx context.Context, key, value []byte) error {
	return d.p.SendMessage(ctx, key, value)
}

func (d *DelayKafka) consumeDelay() error {
	return d.c.Consume([]string{d.delayTopic}, d.proxyGroupID, d.delaySend)
}

func (d *DelayKafka) Consume(groupID string, f func(ctx context.Context, key []byte, value []byte)) error {
	if groupID == d.proxyGroupID {
		return consumer.ErrInvalidGroupID
	}
	handler := consumer.NewFuncConsumeHandler(f, d.log, d.m)
	return d.c.Consume([]string{d.realTopic}, groupID, handler)
}

func (d *DelayKafka) Close() {
	if d.p != nil {
		d.p.Close()
	}
	if d.c != nil {
		d.c.Close()
	}
}
