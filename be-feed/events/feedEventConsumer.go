package events

import (
	"context"
	"errors"
	"fmt"
	"github.com/IBM/sarama"
	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"github.com/asynccnu/ccnubox-be/be-feed/events/consumer"
	"github.com/asynccnu/ccnubox-be/be-feed/events/topic"
	"github.com/asynccnu/ccnubox-be/be-feed/pkg/logger"
	"github.com/asynccnu/ccnubox-be/be-feed/pkg/saramax"
	"github.com/asynccnu/ccnubox-be/be-feed/service"
)

// FeedEventConsumerHandler 是处理 Feed 事件消费的结构体
type FeedEventConsumerHandler struct {
	cg          consumer.Consumer        //消费者
	l           logger.Logger            // 日志记录器
	stopChan    chan struct{}            //用于停止的管道,没用上
	feedService service.FeedEventService // 事件数据的存储库
	pushService service.PushService      //用于推送给客户端的服务

}

// NewFeedEventConsumerHandler 是 FeedEventConsumerHandler 的构造函数
// 接收 Kafka 客户端、日志记录器和事件存储库作为参数，并返回一个 FeedEventConsumerHandler 实例
func NewFeedEventConsumerHandler(kafkaClient sarama.Client, l logger.Logger, feedService service.FeedEventService, pushService service.PushService) *FeedEventConsumerHandler {
	cg := consumer.NewSaramaConsumer(kafkaClient, topic.FeedEvent)
	return &FeedEventConsumerHandler{
		cg:          cg,
		l:           l,
		feedService: feedService,
		stopChan:    make(chan struct{}),
		pushService: pushService,
	}
}

// Start 启动事件消费的流程
func (f *FeedEventConsumerHandler) Start() error {

	// 启动一个 Goroutine 异步消费消息
	go func() {
		// 开始消费主题为 "feed_event" 的消息，并使用自定义的处理函数
		er := f.cg.Consume(context.Background(), []string{topic.FeedEvent}, saramax.NewHandler(f.l, f.Consume))
		if er != nil {
			// 如果消费循环中出现错误，记录错误日志
			f.l.Error("退出了消费循环异常", logger.Error(er))
			//feed消息消费出现问题属于重大问题,选择直接panic
			panic(er)
		}
	}()
	return nil
}

// Consume 是实际处理 Kafka 消息的函数
// 接收 Kafka 消息和事件数组作为参数,并存储到到临时变量里面去
func (f *FeedEventConsumerHandler) Consume(events []domain.FeedEvent) error {
	var ctx = context.Background()

	errs := f.feedService.InsertEventList(ctx, events)
	if errs != nil {
		return errors.Join(errs...)
	}

	errWithData := f.pushService.PushMSGS(ctx, events)
	if len(errWithData) > 0 {
		//类型转换
		failEvent := make([]domain.FeedEvent, len(events))
		for i := range errWithData {
			failEvent[i] = *errWithData[i].FeedEvent
		}

		err := f.pushService.InsertFailFeedEvents(ctx, failEvent)
		if err != nil {
			return err
		}

		return errors.New(fmt.Sprintf("批量消费发生错误! 原数据量为%d,发生错误次数为:%d,首次发生错误为:%s", len(events), len(errWithData), errWithData[0].Err))
	}

	return nil
}
