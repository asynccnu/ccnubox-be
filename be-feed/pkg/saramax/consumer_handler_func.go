package saramax

import (
	"encoding/json"
	"log"
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

	for {
		select {
		//如果接收到了消息，判断是否达到了发送的数量。
		//如果没有达到，就重置定时器，开始计时
		//如果达到了，就停止定时器，发送数据
		case msg := <-claim.Messages():
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
			}
			log.Printf("接收到消息：%v", t)

			// 添加新的值到events中
			events = append(events, t)
			msgRecords = append(msgRecords, msg)
			// 如果数量达到额定值就批量插入消费
			if len(events) >= h.cfg.ConsumeNum {
				log.Printf("进入分支一：数量达到限额")
				e := events[:h.cfg.ConsumeNum]
				err = h.fn(e)
				if err != nil {
					h.l.Error("批量推送消息发生失败", logger.Error(err))
				}

				// 清除插入的数据
				events = []T{}

				//调用fn成功后就标记这些消息消费成功
				for _, m := range msgRecords {
					session.MarkMessage(m, "")
				}
				msgRecords = nil
				//此时队列中的消息消费完，停止计时器
				if !timeout.Stop() {
					select {
					case <-timeout.C:
					default:
					}
				}
			} else {
				log.Printf("进入分支二：数量未达到，开启定时器")
				if !timeout.Stop() {
					select {
					case <-timeout.C:
					default:
					}
				}
				timeout.Reset(time.Minute * time.Duration(h.cfg.ConsumeTime))
			}
		//如果超时，就把未推送的消息推送，定时器停止
		case <-timeout.C:
			log.Printf("超时：消息推送")
			e := events
			err := h.fn(e)
			if err != nil {
				h.l.Error("批量推送消息发生失败", logger.Error(err))
			}

			// 清除插入的数据
			events = []T{}

			//调用fn成功后就标记这些消息消费成功
			for _, m := range msgRecords {
				session.MarkMessage(m, "")
			}
			msgRecords = nil
			timeout.Stop()
		}
	}
}
