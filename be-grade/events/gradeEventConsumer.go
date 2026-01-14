package events

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"
	"github.com/asynccnu/ccnubox-be/be-grade/conf"
	"github.com/asynccnu/ccnubox-be/be-grade/domain"
	"github.com/asynccnu/ccnubox-be/be-grade/events/consumer"
	"github.com/asynccnu/ccnubox-be/be-grade/events/topic"
	"github.com/asynccnu/ccnubox-be/be-grade/service"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/pkg/saramax"
)

// GradeDetailEventConsumerHandler 是处理 GradeDetail 事件消费的结构体
type GradeDetailEventConsumerHandler struct {
	cg           consumer.Consumer    //消费者
	l            logger.Logger        // 日志记录器
	stopChan     chan struct{}        //用于停止的管道,没用上
	gradeService service.GradeService // 事件数据的存储库
	cfg          *saramax.HandlerConfig
}

func NewGradeDetailEventConsumerHandler(
	kafkaClient sarama.Client,
	l logger.Logger,
	gradeService service.GradeService,
	cfg *conf.ServerConf,
) *GradeDetailEventConsumerHandler {
	cg := consumer.NewSaramaConsumer(kafkaClient, topic.GradeDetailEvent)
	return &GradeDetailEventConsumerHandler{
		cg: cg,
		l:  l,
		cfg: &saramax.HandlerConfig{
			ConsumeTime: cfg.ConsumeConf.ConsumeTime,
			ConsumeNum:  cfg.ConsumeConf.ConsumeNum,
		},
		gradeService: gradeService,
		stopChan:     make(chan struct{}),
	}
}

// Start 启动事件消费的流程
func (f *GradeDetailEventConsumerHandler) Start() error {

	// 启动一个 Goroutine 异步消费消息
	go func() {
		for {
			f.l.Info("开始消费")
			err := f.cg.Consume(context.Background(), []string{topic.GradeDetailEvent}, saramax.NewHandler(f.l, f.cfg, f.Consume))
			if err != nil {
				// 如果消费循环中出现错误，记录错误日志
				f.l.Error("退出了消费循环异常", logger.Error(err))
				//feed消息消费出现问题属于重大问题,选择直接panic
				panic(err)
			}
			f.l.Info("消费者退出消费")
		}

	}()
	return nil
}

// Consume 是实际处理 Kafka 消息的函数
// 接收 Kafka 消息和事件数组作为参数,并存储到到临时变量里面去
func (f *GradeDetailEventConsumerHandler) Consume(events []domain.NeedDetailGrade) error {
	var ctx = context.Background()
	for _, event := range events {
		err := f.gradeService.UpdateDetailScore(ctx, event)
		if err != nil {
			f.l.Warn(fmt.Sprintf("更新%s成绩详情失败:", event.StudentID), logger.Error(err))
		}
	}
	return nil
}
