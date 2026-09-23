package consumer

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/pkg/metricsx"
	"github.com/asynccnu/ccnubox-be/common/pkg/otelx/otelsarama"
	"github.com/asynccnu/ccnubox-be/common/pkg/saramax"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// DelaySendHandler 消费延迟 topic消息并转发到真实 topic
type DelaySendHandler struct {
	delayTopic    string
	topic         string
	kp            sarama.SyncProducer
	sendGuard     *saramax.SendGuard
	delayTime     time.Duration
	log           logger.Logger
	setOnce       sync.Once
	downOnce      sync.Once
	producedTotal *prometheus.CounterVec
	consumedTotal *prometheus.CounterVec
	mqFailedTotal *prometheus.CounterVec
}

func NewDelaySendHandler(delayTopic, topic string, client sarama.Client, delayTime time.Duration, l logger.Logger, m *metricsx.Metrics, sendGuard *saramax.SendGuard) (*DelaySendHandler, error) {
	kp, err := sarama.NewSyncProducerFromClient(client)
	if err != nil {
		return nil, err
	}
	return &DelaySendHandler{
		delayTopic:    delayTopic,
		topic:         topic,
		kp:            kp,
		sendGuard:     sendGuard,
		delayTime:     delayTime,
		log:           l,
		producedTotal: m.MQMetrics.ProducedTotal,
		consumedTotal: m.MQMetrics.ConsumedTotal,
		mqFailedTotal: m.MQMetrics.FailedTotal,
	}, nil
}

func (c *DelaySendHandler) Setup(sarama.ConsumerGroupSession) error {
	c.setOnce.Do(func() {
		c.log.Infof("delay send handler setup")
	})
	return nil
}

func (c *DelaySendHandler) Cleanup(sarama.ConsumerGroupSession) error {
	c.downOnce.Do(func() {
		c.log.Infof("delay send handler cleanup")
	})
	return nil
}

func (c *DelaySendHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case <-session.Context().Done():
			return nil
		case message, ok := <-claim.Messages():
			if !ok || session.Context().Err() != nil {
				return nil
			}
			if !c.processMessage(session, message) || session.Context().Err() != nil {
				return nil
			}
			session.MarkMessage(message, "")
		}
	}
}

// processMessage 等待消息到期并转发，失败在当前分区内有上限地退避，不反复重建会话。
// 返回 false 表示 session 已结束，当前消息不得提交 offset。
func (c *DelaySendHandler) processMessage(session sarama.ConsumerGroupSession, message *sarama.ConsumerMessage) bool {
	// 未到期就原地等到期。分区内消息按投递时间有序，阻塞本分区正是延迟队列想要的语义。
	for {
		dur := time.Since(message.Timestamp)
		if dur >= c.delayTime {
			break
		}
		if !sleepWithContext(session.Context(), c.delayTime-dur) {
			return false
		}
	}

	// 严重滞后的消息直接丢弃，不再转发。
	if c.delayTime > 0 && time.Since(message.Timestamp) >= 20*c.delayTime {
		c.log.Warnf(saramax.LogKeyMessageDropped+" delay message expired and dropped: topic=%s partition=%d offset=%d", message.Topic, message.Partition, message.Offset)
		return session.Context().Err() == nil
	}

	var backoff saramax.Backoff
	for attempt := 1; ; attempt++ {
		if session.Context().Err() != nil {
			return false
		}
		ctx := otel.GetTextMapPropagator().Extract(session.Context(), otelsarama.NewConsumerMessageCarrier(message))

		tracer := otel.Tracer("delay-queue-consume")
		ctx, span := tracer.Start(ctx, "delay-queue-consume",
			trace.WithSpanKind(trace.SpanKindConsumer),
		)

		tlog := c.log.WithContext(ctx)
		tlog.Debugf("Message claimed: key:%s, value:%s, time_sub:%v",
			string(message.Key), string(message.Value), time.Since(message.Timestamp))

		err := c.forwardMessage(ctx, message)
		if err == nil {
			// 消费计数
			if c.producedTotal != nil {
				c.producedTotal.WithLabelValues(c.topic, "OK").Inc()
			}
			if c.consumedTotal != nil {
				c.consumedTotal.WithLabelValues(c.delayTopic, "OK").Inc()
			}
			span.End()
			return true
		}

		tlog.Errorf(saramax.LogKeyConsumeRetry+" Error forwarding message: topic=%s partition=%d offset=%d err=%v", message.Topic, message.Partition, message.Offset, err)
		span.RecordError(err)
		if c.mqFailedTotal != nil {
			c.mqFailedTotal.WithLabelValues(c.topic, classifyError(err)).Inc()
		}

		tlog.Warnf(saramax.LogKeyConsumeRetry+" forwarding failed; retaining offset and backing off: topic=%s partition=%d offset=%d attempt=%d",
			message.Topic, message.Partition, message.Offset, attempt)
		span.End()
		logStuckPartition(tlog, attempt, message)
		if backoff.Wait(session.Context()) != nil {
			return false
		}
	}
}

func (c *DelaySendHandler) forwardMessage(ctx context.Context, msg *sarama.ConsumerMessage) error {
	otel.GetTextMapPropagator().Inject(ctx, otelsarama.NewConsumerMessageCarrier(msg))

	tlog := c.log.WithContext(ctx)

	err := c.sendGuard.Send(ctx, func() error {
		_, _, err := c.kp.SendMessage(&sarama.ProducerMessage{
			Topic:   c.topic,
			Key:     sarama.ByteEncoder(msg.Key),
			Value:   sarama.ByteEncoder(msg.Value),
			Headers: consumerHeaders(msg.Headers),
		})
		return err
	})
	if err == nil {
		tlog.Debugf("Forwarded message: key=%s,val=%s,timestamp=%v, current-time=%v", string(msg.Key), string(msg.Value), msg.Timestamp, time.Now())
	}
	return err
}

func consumerHeaders(headers []*sarama.RecordHeader) []sarama.RecordHeader {
	result := make([]sarama.RecordHeader, 0, len(headers))
	for _, header := range headers {
		if header != nil {
			result = append(result, *header)
		}
	}
	return result
}

func (c *DelaySendHandler) Close() error {
	return c.sendGuard.Close(c.kp.Close)
}

// FuncConsumeHandler 消费真实 topic 消息并交付给应用
type FuncConsumeHandler struct {
	f             func(ctx context.Context, key []byte, value []byte) (ack bool, err error)
	log           logger.Logger
	consumedTotal *prometheus.CounterVec
	mqFailedTotal *prometheus.CounterVec
}

func NewFuncConsumeHandler(f func(ctx context.Context, key []byte, value []byte) (ack bool, err error), l logger.Logger, m *metricsx.Metrics) FuncConsumeHandler {
	return FuncConsumeHandler{
		f:             f,
		log:           l,
		consumedTotal: m.MQMetrics.ConsumedTotal,
		mqFailedTotal: m.MQMetrics.FailedTotal,
	}
}

func (fc FuncConsumeHandler) Setup(sarama.ConsumerGroupSession) error {
	fc.log.Info("Setting up func consume handler")
	return nil
}

func (fc FuncConsumeHandler) Cleanup(sarama.ConsumerGroupSession) error {
	fc.log.Info("Cleaning up func consume handler")
	return nil
}

func (fc FuncConsumeHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case <-session.Context().Done():
			return nil
		case message, ok := <-claim.Messages():
			if !ok || session.Context().Err() != nil {
				return nil
			}
			if !fc.handleWithRetry(session, message) || session.Context().Err() != nil {
				return nil
			}
			// 下一条重试已可靠发布，或业务明确终止时，才允许确认。
			session.MarkMessage(message, "")
		}
	}
}

// handleWithRetry 尊重业务 ack=false，始终保留当前消息并退避，不越过失败位点。
// 返回 false 表示 session 已结束，当前消息不得提交 offset。
func (fc FuncConsumeHandler) handleWithRetry(session sarama.ConsumerGroupSession, message *sarama.ConsumerMessage) bool {
	var backoff saramax.Backoff
	for attempt := 1; ; attempt++ {
		if session.Context().Err() != nil {
			return false
		}
		ctx := otel.GetTextMapPropagator().Extract(session.Context(), otelsarama.NewConsumerMessageCarrier(message))

		tracer := otel.Tracer("real-topic")
		ctx, span := tracer.Start(ctx, "real_topic_consumer",
			trace.WithSpanKind(trace.SpanKindConsumer),
		)

		tlog := fc.log.WithContext(ctx)

		tlog.Debugf("Message claimed: key:%s, value:%s", string(message.Key), string(message.Value))
		ack, err := fc.f(ctx, message.Key, message.Value)
		if !ack && err == nil {
			err = errors.New("message requested retry without an error")
		}
		if err != nil {
			tlog.Errorf("Error handling message: %v", err)
			span.RecordError(err)
			if fc.consumedTotal != nil {
				fc.consumedTotal.WithLabelValues(message.Topic, "Error").Inc()
			}
			if fc.mqFailedTotal != nil {
				fc.mqFailedTotal.WithLabelValues(message.Topic, classifyError(err)).Inc()
			}
		} else if fc.consumedTotal != nil {
			fc.consumedTotal.WithLabelValues(message.Topic, "OK").Inc()
		}
		span.End()

		if ack {
			return true
		}
		tlog.Warnf(saramax.LogKeyConsumeRetry+" message not acknowledged; retaining offset and backing off: topic=%s partition=%d offset=%d attempt=%d",
			message.Topic, message.Partition, message.Offset, attempt)
		logStuckPartition(tlog, attempt, message)
		if backoff.Wait(session.Context()) != nil {
			return false
		}
	}
}

// blockedAttemptThreshold 是“分区已长时间停摆”的判定阈值。
// 达到该次数后按关键字报错，便于在没有告警的情况下定位需要人工介入的消息。
const blockedAttemptThreshold = 10

// logStuckPartition 在单条消息反复失败时输出可检索的阻塞日志。
func logStuckPartition(tlog logger.Logger, attempt int, message *sarama.ConsumerMessage) {
	if attempt != blockedAttemptThreshold {
		return
	}
	tlog.Errorf(saramax.LogKeyPartitionBlocked+" 消息持续失败，该分区在人工处理前不会前进: topic=%s partition=%d offset=%d attempt=%d",
		message.Topic, message.Partition, message.Offset, attempt)
}

// sleepWithContext 用于等待消息到期，rebalance 或关闭时立即停止。
func sleepWithContext(ctx context.Context, d time.Duration) bool {
	if ctx.Err() != nil {
		return false
	}
	timer := time.NewTimer(max(d, 0))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return ctx.Err() == nil
	}
}

type Consumer struct {
	ctx        context.Context // 根 context：Close 时取消，兼作调用方未提供 context 时的兼容
	cancelFunc context.CancelFunc
	newClient  ClientFactory
	newGroup   consumerGroupFactory
	log        logger.Logger
	mu         sync.Mutex
	wg         sync.WaitGroup
}

type ClientFactory func() (sarama.Client, error)

type consumerGroupFactory func(groupID string, client sarama.Client) (sarama.ConsumerGroup, error)

func NewConsumer(newClient ClientFactory, l logger.Logger) *Consumer {
	ctx, cancel := context.WithCancel(context.Background())
	return &Consumer{
		ctx:        ctx,
		cancelFunc: cancel,
		newClient:  newClient,
		newGroup:   sarama.NewConsumerGroupFromClient,
		log:        l,
	}
}

// Consume 用调用方传入的 ctx 控制消费循环的生命周期：调用方 ctx 或 Close 任一取消都会停止消费。
// 新会话失败后按退避重建，不退出后台协程。
func (c *Consumer) Consume(ctx context.Context, topics []string, groupID string, handler sarama.ConsumerGroupHandler) error {
	if c.newClient == nil {
		return ErrNilClient
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	// Close 时也要能结束本轮消费。
	stop := context.AfterFunc(c.ctx, cancel)
	defer stop()

	c.mu.Lock()
	if c.ctx.Err() != nil {
		c.mu.Unlock()
		return c.ctx.Err()
	}
	c.wg.Add(1)
	c.mu.Unlock()
	defer c.wg.Done()

	var backoff saramax.Backoff
	for runCtx.Err() == nil {
		err := c.consumeGroup(runCtx, topics, groupID, handler)
		if runCtx.Err() != nil {
			return runCtx.Err()
		}
		c.log.Error(saramax.LogKeyConsumeRetry+" 消费组创建或运行失败，保留任务并退避重建", logger.String("group_id", groupID), logger.Error(err))
		if err := backoff.Wait(runCtx); err != nil {
			return err
		}
	}
	return runCtx.Err()
}

func (c *Consumer) consumeGroup(ctx context.Context, topics []string, groupID string, handler sarama.ConsumerGroupHandler) error {
	// 不共用 Client；创建失败也交给外层恢复，不退出后台协程。
	client, err := c.newClient()
	if err != nil {
		return err
	}
	if client == nil {
		return ErrNilClient
	}
	// 消费者组由该 client 创建，cg.Close() 会一并关闭 client（sarama 语义），
	// 这里只兜住建组失败的情况，重复关闭不算错误。
	defer func() {
		if err := client.Close(); err != nil && !errors.Is(err, sarama.ErrClosedClient) {
			c.log.Errorf("Error closing Kafka client for consumer group %s: %v", groupID, err)
		}
	}()
	if err := ctx.Err(); err != nil {
		return err
	}
	cg, err := c.newGroup(groupID, client)
	if err != nil {
		return err
	}
	defer func() {
		if err := cg.Close(); err != nil {
			c.log.Errorf("Error closing consumer group %s: %v", groupID, err)
		}
	}()
	return saramax.RunConsumer(ctx, cg, topics, handler, c.log)
}

func (c *Consumer) Close() {
	c.mu.Lock()
	c.cancelFunc()
	c.mu.Unlock()
	// 等待会话释放位点和 Client，再由调用方关闭转发生产者。
	c.wg.Wait()
}

var (
	ErrInvalidGroupID = errors.New("the groupID is not allowed")
	ErrNilClient      = errors.New("kafka client factory returned nil")
)

// classifyError 将 Kafka/Sarama 错误分类，用于 mq_failed_total 标签
func classifyError(err error) string {
	var producerErr *sarama.ProducerError
	if errors.As(err, &producerErr) {
		err = producerErr.Err
	}

	if errors.Is(err, sarama.ErrLeaderNotAvailable) {
		return "leader_not_available"
	}
	if errors.Is(err, sarama.ErrNotEnoughReplicas) || errors.Is(err, sarama.ErrNotEnoughReplicasAfterAppend) {
		return "not_enough_replicas"
	}
	if errors.Is(err, sarama.ErrMessageTooLarge) || errors.Is(err, sarama.ErrMessageSizeTooLarge) {
		return "message_too_large"
	}
	if errors.Is(err, sarama.ErrInvalidTopic) {
		return "invalid_topic"
	}
	return "consume_error"
}
