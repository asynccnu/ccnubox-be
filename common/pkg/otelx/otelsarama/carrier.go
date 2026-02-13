package otelsarama

import (
	"github.com/IBM/sarama"
	"go.opentelemetry.io/otel/propagation"
)

type ProducerMessageCarrier struct {
	msg *sarama.ProducerMessage
}

func NewProducerMessageCarrier(msg *sarama.ProducerMessage) propagation.TextMapCarrier {
	return &ProducerMessageCarrier{msg: msg}
}

func (c *ProducerMessageCarrier) Get(key string) string {
	for _, h := range c.msg.Headers {
		if string(h.Key) == key {
			return string(h.Value)
		}
	}
	return ""
}

func (c *ProducerMessageCarrier) Set(key string, value string) {
	// 创建要注入的 header 捏
	newHeader := sarama.RecordHeader{
		Key:   []byte(key),
		Value: []byte(value),
	}

	// 防止重复注入同一个 key
	for i, h := range c.msg.Headers {
		if string(h.Key) == key {
			// 找到直接覆盖
			c.msg.Headers[i].Value = []byte(value)
			return
		}
	}

	c.msg.Headers = append(c.msg.Headers, newHeader)
}

func (c *ProducerMessageCarrier) Keys() []string {
	out := make([]string, 0, len(c.msg.Headers))
	for _, h := range c.msg.Headers {
		out = append(out, string(h.Key))
	}

	return out
}

type ConsumerMessageCarrier struct {
	msg *sarama.ConsumerMessage
}

func NewConsumerMessageCarrier(msg *sarama.ConsumerMessage) propagation.TextMapCarrier {
	return &ConsumerMessageCarrier{msg: msg}
}

func (c *ConsumerMessageCarrier) Get(key string) string {
	for _, h := range c.msg.Headers {
		if string(h.Key) == key {
			return string(h.Value)
		}
	}
	return ""
}

func (c *ConsumerMessageCarrier) Set(key string, value string) {
	// 创建要注入的 header 捏
	newHeader := &sarama.RecordHeader{
		Key:   []byte(key),
		Value: []byte(value),
	}

	// 防止重复注入同一个 key
	for i, h := range c.msg.Headers {
		if string(h.Key) == key {
			// 找到直接覆盖
			c.msg.Headers[i].Value = []byte(value)
			return
		}
	}

	c.msg.Headers = append(c.msg.Headers, newHeader)
}

func (c *ConsumerMessageCarrier) Keys() []string {
	out := make([]string, 0, len(c.msg.Headers))
	for _, h := range c.msg.Headers {
		out = append(out, string(h.Key))
	}

	return out
}
