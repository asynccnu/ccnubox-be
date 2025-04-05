package saramax

import (
	"encoding/json"
	"github.com/IBM/sarama"
	"github.com/asynccnu/ccnubox-be/be-feed/pkg/logger"
	"github.com/spf13/viper"
	"time"
)

type MSG struct {
	Topic     string
	Partition int32
	Offset    int64
}

type Handler[T any] struct {
	l   logger.Logger
	fn  func(t []T) error
	cfg HandlerConfig
}

type HandlerConfig struct {
	ConsumeTime int `yaml:"consumeTime"`
	ConsumeNum  int `yaml:"consumeNum"`
}

func NewHandler[T any](l logger.Logger,
	fn func(t []T) error) *Handler[T] {
	var cfg HandlerConfig
	if err := viper.UnmarshalKey("consume", &cfg); err != nil {
		panic(err)
	}
	return &Handler[T]{
		l:   l,
		fn:  fn,
		cfg: cfg,
	}
}

func (h *Handler[T]) Setup(session sarama.ConsumerGroupSession) error {
	return nil
}

func (h *Handler[T]) Cleanup(session sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim 可以考虑在这个封装里面提供统一的重试机制
func (h *Handler[T]) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	var events []T
	var msgRecords []MSG
	var lastConsumeTime time.Time // 记录上次消费的时间

	msgs := claim.Messages()

	for msg := range msgs {
		// 从msg中提取获得附带的值
		var t T
		err := json.Unmarshal(msg.Value, &t)
		if err != nil {
			h.l.Error("反序列化消息体失败",
				logger.String("topic", msg.Topic),
				logger.Int32("partition", msg.Partition),
				logger.Int64("offset", msg.Offset),
				logger.Error(err))
			session.MarkMessage(msg, "")
			continue
		}

		// 添加新的值到events中
		events = append(events, t)
		msgRecords = append(msgRecords, MSG{Topic: msg.Topic, Partition: msg.Partition, Offset: msg.Offset})
		// 如果数量达到额定值就批量插入消费
		if len(events) >= h.cfg.ConsumeNum {
			e := events[:h.cfg.ConsumeNum]
			err = h.fn(e)
			if err != nil {
				h.l.Error("批量推送消息发生失败", logger.Error(err))
			}

			// 更新上次消费时间
			lastConsumeTime = time.Now()
			// 清除插入的数据
			events = events[h.cfg.ConsumeNum:]
			msgRecords = msgRecords[h.cfg.ConsumeNum:]
		} else if !lastConsumeTime.IsZero() && time.Since(lastConsumeTime) > time.Duration(h.cfg.ConsumeTime)*time.Minute {
			// 如果距离上次消费已经超过5分钟且有未处理的消息
			if len(events) > 0 {
				e := events[:]
				err = h.fn(e)
				if err != nil {
					h.l.Error("批量推送消息发生失败", logger.Error(err))
				}
				// 更新上次消费时间
				lastConsumeTime = time.Now()
				// 清空events和msgRecords
				events = []T{}
				msgRecords = []MSG{}
			}
		}
		session.MarkMessage(msg, "")
	}
	return nil
}
