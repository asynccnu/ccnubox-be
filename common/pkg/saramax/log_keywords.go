package saramax

// 没有 DLQ 和告警的情况下，坏消息与发送失败只能靠日志发现。
// 这里统一几个固定关键字，拼在日志消息最前面，便于 grep 或日志平台按关键字检索/告警：
//
//	KAFKA_PARTITION_BLOCKED 位点未确认、分区停止消费，必须人工介入（坏消息或依赖长期不可用）。
//	KAFKA_CONSUME_RETRY     本轮处理失败，正在退避重试，通常可以自愈。
//	KAFKA_MESSAGE_DROPPED   消息被显式丢弃，不会再重投（永久失败达到阈值、过期或业务终态）。
//	KAFKA_SEND_FAILED       消息没有进入 Kafka（发送失败，或冷却期内被直接拒绝）。
//	KAFKA_CONSUMER_PANIC    消费逻辑 panic，已兜底成一次处理失败（不会崩进程）。
const (
	LogKeyPartitionBlocked = "KAFKA_PARTITION_BLOCKED"
	LogKeyConsumeRetry     = "KAFKA_CONSUME_RETRY"
	LogKeyMessageDropped   = "KAFKA_MESSAGE_DROPPED"
	LogKeySendFailed       = "KAFKA_SEND_FAILED"
	LogKeyConsumerPanic    = "KAFKA_CONSUMER_PANIC"
)
