package infra

import (
	"log"
	"time"

	"github.com/IBM/sarama"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/conf"
)

type KafkaConfigOption func(*sarama.Config)

// InitKafka 面向启动期必须连上 Kafka 的服务（如 be-grade）：初始化失败直接退出进程。
// 若希望初始化失败交给后台消费循环退避重试，改用 NewKafka。
func InitKafka(cfg *conf.KafkaConf, options ...KafkaConfigOption) sarama.Client {
	client, err := NewKafka(cfg, options...)
	if err != nil {
		log.Fatal("初始化 kafka 失败", err)
	}
	return client
}

// NewKafka 将初始化错误返回给调用方，供需要退避恢复的后台消费者使用
// （如 be-classlist 的消费循环），不在恢复循环中退出进程。
func NewKafka(cfg *conf.KafkaConf, options ...KafkaConfigOption) (sarama.Client, error) {
	saramaCfg := sarama.NewConfig()

	// 限制单次失败的网络等待和内部重试，不与业务重试层叠放大。
	saramaCfg.Net.DialTimeout = 3 * time.Second
	saramaCfg.Net.ReadTimeout = saramaCfg.Consumer.Group.Rebalance.Timeout + 10*time.Second
	saramaCfg.Net.WriteTimeout = 5 * time.Second
	saramaCfg.Producer.RequiredAcks = sarama.WaitForAll
	saramaCfg.Producer.Timeout = 5 * time.Second
	saramaCfg.Producer.Retry.Max = 1
	saramaCfg.Producer.Retry.Backoff = time.Second
	saramaCfg.Metadata.Retry.Backoff = time.Second
	// 保持已有位点与新组的起点策略，不在升级时自动触发历史全量回放。
	saramaCfg.Consumer.Offsets.Initial = sarama.OffsetNewest

	saramaCfg.Net.SASL.Enable = true
	saramaCfg.Net.SASL.User = cfg.Username
	saramaCfg.Net.SASL.Password = cfg.Password
	saramaCfg.Net.SASL.Mechanism = sarama.SASLTypePlaintext

	saramaCfg.Producer.Return.Successes = true
	saramaCfg.Producer.Partitioner = sarama.NewConsistentCRCHashPartitioner
	for _, option := range options {
		if option != nil {
			option(saramaCfg)
		}
	}

	// JoinGroup 使用通用网络读超时；必须覆盖 broker 等待成员重加入的窗口。
	saramaCfg.Net.ReadTimeout = max(saramaCfg.Net.ReadTimeout, saramaCfg.Consumer.Group.Rebalance.Timeout+10*time.Second)
	return sarama.NewClient(cfg.Addrs, saramaCfg)
}
