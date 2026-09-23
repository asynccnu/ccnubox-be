package events

import (
	"context"
	"errors"
	"sync"

	"github.com/IBM/sarama"
	"github.com/asynccnu/ccnubox-be/be-grade/conf"
	"github.com/asynccnu/ccnubox-be/be-grade/domain"
	"github.com/asynccnu/ccnubox-be/be-grade/events/consumer"
	"github.com/asynccnu/ccnubox-be/be-grade/events/topic"
	"github.com/asynccnu/ccnubox-be/be-grade/service"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/pkg/metricsx"
	"github.com/asynccnu/ccnubox-be/common/pkg/saramax"
)

// GradeDetailEventConsumerHandler 是处理 GradeDetail 事件消费的结构体
type GradeDetailEventConsumerHandler struct {
	cg           consumer.Consumer    //消费者
	l            logger.Logger        // 日志记录器
	gradeService service.GradeService // 事件数据的存储库
	cfg          *saramax.HandlerConfig
	m            *metricsx.Metrics
	ctx          context.Context
	cancel       context.CancelFunc
	stopOnce     sync.Once
	wg           sync.WaitGroup
}

func NewGradeDetailEventConsumerHandler(
	kafkaClient sarama.Client,
	l logger.Logger,
	gradeService service.GradeService,
	cfg *conf.ServerConf,
	m *metricsx.Metrics,
) *GradeDetailEventConsumerHandler {
	cg := consumer.NewSaramaConsumer(kafkaClient, topic.GradeDetailEvent)
	ctx, cancel := context.WithCancel(context.Background())
	return &GradeDetailEventConsumerHandler{
		cg: cg,
		l:  l,
		cfg: &saramax.HandlerConfig{
			ConsumeTime: cfg.ConsumeConf.ConsumeTime,
			ConsumeNum:  cfg.ConsumeConf.ConsumeNum,
		},
		gradeService: gradeService,
		ctx:          ctx,
		cancel:       cancel,
		m:            m,
	}
}

// Start 启动事件消费的流程
func (f *GradeDetailEventConsumerHandler) Start() error {

	// 启动一个 Goroutine 异步消费消息
	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		saramax.RunConsumer(f.ctx, f.cg, []string{topic.GradeDetailEvent},
			saramax.NewContextHandler(f.l, f.cfg, f.consume), f.l)
	}()
	return nil
}

func (f *GradeDetailEventConsumerHandler) Stop() {
	f.stopOnce.Do(func() {
		if f.cancel != nil {
			f.cancel()
		}
		if f.cg != nil {
			if err := f.cg.Close(); err != nil {
				f.l.Error("close grade consumer failed", logger.Error(err))
			}
		}
	})
	f.wg.Wait()
}

// Consume 是实际处理 Kafka 消息的函数
// 接收 Kafka 消息和事件数组作为参数,并存储到到临时变量里面去
func (f *GradeDetailEventConsumerHandler) Consume(events []domain.NeedDetailGrade) error {
	return f.consume(context.Background(), events)
}

func (f *GradeDetailEventConsumerHandler) consume(ctx context.Context, events []domain.NeedDetailGrade) error {
	var failed int
	var consumeErrors []error
	for _, event := range events {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := f.gradeService.UpdateDetailScore(ctx, event)
		if err != nil {
			// 单个学生的详情失败会让整批重投并结束本轮会话，需要靠关键字定位是哪个学生一直失败，
			// 具体失败科目由 service 层日志给出（不打印成绩等隐私内容）。
			f.l.Warn(saramax.LogKeyConsumeRetry+" 更新成绩详情失败，整批将重投",
				logger.String("sid", event.StudentID), logger.Int("grades", len(event.Grades)), logger.Error(err))
			failed++
			consumeErrors = append(consumeErrors, err)
		}
	}
	if f.m != nil && f.m.MQMetrics != nil {
		if failed > 0 && f.m.MQMetrics.FailedTotal != nil {
			f.m.MQMetrics.FailedTotal.WithLabelValues(topic.GradeDetailEvent, "consume_error").Add(float64(failed))
		}
		if f.m.MQMetrics.ConsumedTotal != nil {
			f.m.MQMetrics.ConsumedTotal.WithLabelValues(topic.GradeDetailEvent, "OK").Add(float64(len(events) - failed))
		}
	}
	return errors.Join(consumeErrors...)
}
