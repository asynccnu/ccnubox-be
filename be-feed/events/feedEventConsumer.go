package events

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"github.com/IBM/sarama"
	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"github.com/asynccnu/ccnubox-be/be-feed/events/consumer"
	"github.com/asynccnu/ccnubox-be/be-feed/events/topic"
	"github.com/asynccnu/ccnubox-be/be-feed/service"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/pkg/metricsx"
	"github.com/asynccnu/ccnubox-be/common/pkg/saramax"
)

// FeedEventConsumerHandler 是处理 Feed 事件消费的结构体
type FeedEventConsumerHandler struct {
	cg          consumer.Consumer        //消费者
	l           logger.Logger            // 日志记录器
	feedService service.FeedEventService // 事件数据的存储库
	m           *metricsx.Metrics
	sk          *saramax.Skipper // 永久失败消息的跳过判定
	ctx         context.Context
	cancel      context.CancelFunc
	stopOnce    sync.Once
	wg          sync.WaitGroup
}

// NewFeedEventConsumerHandler 是 FeedEventConsumerHandler 的构造函数
// 接收 Kafka 客户端、日志记录器和事件存储库作为参数，并返回一个 FeedEventConsumerHandler 实例
func NewFeedEventConsumerHandler(
	kafkaClient sarama.Client,
	l logger.Logger,
	feedService service.FeedEventService,
	m *metricsx.Metrics,
) *FeedEventConsumerHandler {
	cg := consumer.NewSaramaConsumer(kafkaClient, topic.FeedEvent)
	ctx, cancel := context.WithCancel(context.Background())
	return &FeedEventConsumerHandler{
		cg:          cg,
		l:           l,
		feedService: feedService,
		m:           m,
		sk:          saramax.NewSkipper(saramax.DefaultSkipAttempts, l),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start 启动事件消费的流程
func (f *FeedEventConsumerHandler) Start() error {

	// 启动一个 Goroutine 异步消费消息
	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		saramax.RunConsumer(f.ctx, f.cg, []string{topic.FeedEvent}, &feedEventKafkaHandler{consumer: f}, f.l)
	}()
	return nil
}

func (f *FeedEventConsumerHandler) Stop() {
	f.stopOnce.Do(func() {
		if f.cancel != nil {
			f.cancel()
		}
		if f.cg != nil {
			if err := f.cg.Close(); err != nil {
				f.l.Error("close feed consumer failed", logger.Error(err))
			}
		}
	})
	f.wg.Wait()
}

func (f *FeedEventConsumerHandler) recordFailure(errorType string, count int) {
	if count <= 0 || f.m == nil || f.m.MQMetrics == nil || f.m.MQMetrics.FailedTotal == nil {
		return
	}
	f.m.MQMetrics.FailedTotal.WithLabelValues(topic.FeedEvent, errorType).Add(float64(count))
}

func (f *FeedEventConsumerHandler) recordConsumed(status string, count int) {
	if count <= 0 || f.m == nil || f.m.MQMetrics == nil || f.m.MQMetrics.ConsumedTotal == nil {
		return
	}
	f.m.MQMetrics.ConsumedTotal.WithLabelValues(topic.FeedEvent, status).Add(float64(count))
}

// feedEventKafkaHandler 按分区顺序处理消息。任何失败都不确认位点，
// 也不再处理该分区后续消息，避免后续确认覆盖失败消息。
type feedEventKafkaHandler struct {
	consumer *FeedEventConsumerHandler
}

func (h *feedEventKafkaHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *feedEventKafkaHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h *feedEventKafkaHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	// claim 跑在 sarama 自己的协程里，panic 会直接崩掉进程，这里兜底成一次普通失败。
	return saramax.CatchPanic(h.consumer.l, func() error {
		return h.consumeClaim(session, claim)
	})
}

func (h *feedEventKafkaHandler) consumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case <-session.Context().Done():
			return nil
		case message, ok := <-claim.Messages():
			if !ok {
				return nil
			}
			if session.Context().Err() != nil {
				return nil
			}
			fields := saramax.MessageFields(message)
			// panic 在这里兜底，否则会连带崩掉整个进程。
			ack, err := saramax.Catch(h.consumer.l, func() (bool, error) {
				return h.consumeMessage(session.Context(), message)
			}, fields...)
			if err == nil {
				if ack {
					h.consumer.sk.Forget(message)
					session.MarkMessage(message, "")
				}
				continue
			}
			// 永久失败（消息体或字段非法）按阈值跳过，避免一条坏消息永久卡住分区；
			// 依赖故障（DB、下游）不计数也不跳过，保留位点等重投。
			if !saramax.IsPermanent(err) || session.Context().Err() != nil {
				return err
			}
			if err := h.dropPermanentMessage(session, message, fields, err); err != nil {
				return err
			}
		}
	}
}

// dropPermanentMessage 处理反复永久失败的消息：阈值内保留位点并结束本轮会话（留人工介入窗口），
// 达到阈值后跳过它，让分区继续前进。返回非 nil 表示未达阈值，需要停止消费该分区。
func (h *feedEventKafkaHandler) dropPermanentMessage(session sarama.ConsumerGroupSession, message *sarama.ConsumerMessage, fields []logger.Field, cause error) error {
	logh := h.consumer.l.WithContext(session.Context()).With(fields...)
	count, reached := h.consumer.sk.Fail(message)
	if !reached {
		logh.Error(saramax.LogKeyPartitionBlocked+" feed event permanently failed; offset left uncommitted",
			logger.Int("attempts", count), logger.Error(cause))
		return cause
	}
	logh.Error(saramax.LogKeyMessageDropped+" feed event permanently failed and reached skip threshold; dropped",
		logger.Int("attempts", count), logger.Error(cause))
	h.consumer.recordFailure("poison_dropped", 1)
	h.consumer.sk.Forget(message)
	session.MarkMessage(message, "")
	return nil
}

func (h *feedEventKafkaHandler) consumeMessage(ctx context.Context, message *sarama.ConsumerMessage) (bool, error) {
	logh := h.consumer.l.WithContext(ctx)
	fields := []logger.Field{
		logger.String("topic", message.Topic),
		logger.Int32("partition", message.Partition),
		logger.Int64("offset", message.Offset),
		logger.Int("payload_size", len(message.Value)),
	}

	var event domain.FeedEvent
	if err := json.Unmarshal(message.Value, &event); err != nil {
		h.consumer.recordFailure("decode_error", 1)
		logh.Error("feed event invalid payload", append(fields, logger.Error(err))...)
		// 消息体非法是永久失败：由消费循环按阈值决定保留位点还是跳过。
		return false, saramax.Permanent(err)
	}
	if strings.TrimSpace(event.DedupeKey) == "" {
		// 旧生产者没有消息 ID 时使用 Kafka 坐标兜底，同一条消息重投仍得到相同 key。
		event.DedupeKey = domain.KafkaFeedEventDedupeKey(message.Topic, message.Partition, message.Offset, event.StudentId)
	}
	if err := domain.ValidateFeedEventForStorage(event); err != nil {
		h.consumer.recordFailure("invalid_event", 1)
		logh.Error("feed event invalid storage fields", append(fields, logger.Error(err))...)
		return false, saramax.Permanent(err)
	}

	attempts := 0
	consumeErr := saramax.Retry(ctx, logh.With(fields...), 0, func() error {
		attempts++
		if attempts > 1 {
			h.consumer.recordFailure("db_retry", 1)
		}
		return h.consumer.feedService.InsertEventList(ctx, []domain.FeedEvent{event})
	})
	if consumeErr == nil {
		h.consumer.recordConsumed("OK", 1)
		return true, nil
	}
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	h.consumer.recordFailure("db_error", 1)
	logh.Error(saramax.LogKeyPartitionBlocked+" feed event storage retries exhausted; offset left uncommitted",
		append(fields, logger.Int("attempts", attempts), logger.Error(consumeErr))...)
	return false, consumeErr
}
