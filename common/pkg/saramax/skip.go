package saramax

import (
	"errors"
	"sync"

	"github.com/IBM/sarama"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

// PermanentError 标记“重投也不会成功”的失败：消息体解析失败、字段非法、
// 上游写入的数据本身不合法等。只有这类错误参与跳过计数，
// 依赖故障（DB、网络、下游服务）必须返回普通错误，消息会一直保留位点，
// 不会因为故障持续时间变长而被自动丢弃。
type PermanentError struct {
	err error
}

func (e *PermanentError) Error() string { return e.err.Error() }

func (e *PermanentError) Unwrap() error { return e.err }

// Permanent 把 err 标记为永久失败，err 为 nil 时原样返回。
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return &PermanentError{err: err}
}

// IsPermanent 判断错误是否被标记为永久失败（含包装后的错误）。
func IsPermanent(err error) bool {
	var pe *PermanentError
	return errors.As(err, &pe)
}

// ProducerError 将消息大小、topic 非法等确定性发送错误标记为永久失败。
// Sarama 的 ProducerError 不保证通过 Unwrap 暴露底层错误，需要显式检查。
func ProducerError(err error) error {
	if err == nil || IsPermanent(err) {
		return err
	}
	cause := err
	var pe *sarama.ProducerError
	if errors.As(err, &pe) {
		cause = pe.Err
	}
	if errors.Is(cause, sarama.ErrMessageTooLarge) || errors.Is(cause, sarama.ErrMessageSizeTooLarge) || errors.Is(cause, sarama.ErrInvalidTopic) {
		return Permanent(err)
	}
	return err
}

// DefaultSkipAttempts 是一条永久失败的消息连续失败多少次后允许跳过。
// 没有 DLQ 和告警，阈值内的重投是留给人工介入的窗口；超过阈值后丢弃，
// 让分区继续前进——分区长期停摆时积压消息会在 topic 保留期内被 Kafka 清理，损失更大。
const DefaultSkipAttempts = 5

// maxSkipEntries 限制计数键的数量：分区推进后计数会被清理，
// 但长期运行仍可能有残留键，超过上限时整体清零重新累计（只会多试几轮，不会丢消息）。
const maxSkipEntries = 10000

type msgKey struct {
	topic     string
	partition int32
	offset    int64
}

func newMsgKey(msg *sarama.ConsumerMessage) msgKey {
	return msgKey{topic: msg.Topic, partition: msg.Partition, offset: msg.Offset}
}

// Skipper 按位点记录永久失败的次数，用于判定一条消息是否已经被反复证明处理不了。
// 计数只放在内存里：进程重启后重新累计，宁可多试几轮也不轻易丢消息。
// 阈值小于 0 时构造函数返回 nil，表示禁用跳过（消息永久阻塞分区，等人工处理）。
type Skipper struct {
	l         logger.Logger
	threshold int

	mu       sync.Mutex
	failures map[msgKey]int
}

// NewSkipper 创建跳过判定器：threshold 为 0 时使用 DefaultSkipAttempts，
// 小于 0 表示禁用跳过。
func NewSkipper(threshold int, l logger.Logger) *Skipper {
	if threshold < 0 {
		return nil
	}
	if threshold == 0 {
		threshold = DefaultSkipAttempts
	}
	return &Skipper{
		l:         l,
		threshold: threshold,
		failures:  make(map[msgKey]int),
	}
}

// Fail 记录一次永久失败，返回累计次数以及是否达到跳过阈值。
func (s *Skipper) Fail(msg *sarama.ConsumerMessage) (int, bool) {
	if s == nil || msg == nil {
		return 0, false
	}
	key := newMsgKey(msg)
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.failures) >= maxSkipEntries {
		if _, ok := s.failures[key]; !ok {
			// 只在需要新增键时清理，避免正常消费反复触发清空。
			s.failures = make(map[msgKey]int)
			if s.l != nil {
				s.l.Warn("KAFKA 永久失败计数超过上限已重置，跳过阈值重新累计")
			}
		}
	}
	s.failures[key]++
	count := s.failures[key]
	return count, count >= s.threshold
}

// Forget 清理已经确认（处理成功或已跳过）的消息计数。
func (s *Skipper) Forget(msg *sarama.ConsumerMessage) {
	if s == nil || msg == nil {
		return
	}
	key := newMsgKey(msg)
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.failures, key)
}
