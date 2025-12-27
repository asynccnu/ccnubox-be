package saramax

import (
	"encoding/json"
	"time"

	"github.com/IBM/sarama"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/spf13/viper"
)

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
	var msgRecords []*sarama.ConsumerMessage //记录kafka中还未消费的消息

	//超时机制：每次接收到消息就重新开启一个计时，超过五分钟没有接收消息就直接发布
	timeout := time.NewTimer(time.Minute * time.Duration(h.cfg.ConsumeTime))
	timeout.Stop()

	defer func() {
		timeout.Stop()
		if len(events) > 0 {
			h.ConsumeEvents(&events, &msgRecords, session)
		}
	}()

	for {
		select {
		case <-session.Context().Done():
			return nil

		case msg, ok := <-claim.Messages():
			if !ok {
				return nil
			}
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

			events = append(events, t)
			msgRecords = append(msgRecords, msg)
			// 如果数量达到额定值就批量插入消费
			if len(events) >= h.cfg.ConsumeNum {
				h.ConsumeEvents(&events, &msgRecords, session)
				//此时队列中的消息消费完，停止计时器
				h.StopTimer(timeout)
			} else {
				//没有达到消息限额，重置定时器
				h.StopTimer(timeout)
				timeout.Reset(time.Minute * time.Duration(h.cfg.ConsumeTime))
			}

		//如果超时，就把未推送的消息推送，定时器停止
		case <-timeout.C:
			h.ConsumeEvents(&events, &msgRecords, session)
			h.StopTimer(timeout)
		}
	}
}

func (h *Handler[T]) StopTimer(timeout *time.Timer) {
	if !timeout.Stop() {
		select {
		case <-timeout.C:
		default:
		}
	}
}

func (h *Handler[T]) ConsumeEvents(events *[]T, msgRecords *[]*sarama.ConsumerMessage, session sarama.ConsumerGroupSession) {
	if len(*events) == 0 {
		return
	}
	//处理待消费的事件
	err := h.fn(*events)
	if err != nil {
		h.l.Error("批量推送消息发生失败", logger.Error(err))
	}

	//标记消息已消费，清空未消费消息
	for _, msg := range *msgRecords {
		session.MarkMessage(msg, "")
	}

	*events = nil
	*msgRecords = nil
}
