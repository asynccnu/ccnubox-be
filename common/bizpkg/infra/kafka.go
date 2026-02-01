package infra

import (
	"log"

	"github.com/IBM/sarama"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/conf"
)

func InitKafka(cfg *conf.KafkaConf) sarama.Client {
	saramaCfg := sarama.NewConfig()

	saramaCfg.Net.SASL.Enable = true
	saramaCfg.Net.SASL.User = cfg.Username
	saramaCfg.Net.SASL.Password = cfg.Password
	saramaCfg.Net.SASL.Mechanism = sarama.SASLTypePlaintext

	saramaCfg.Producer.Return.Successes = true
	saramaCfg.Producer.Partitioner = sarama.NewConsistentCRCHashPartitioner
	client, err := sarama.NewClient(cfg.Addrs, saramaCfg)
	if err != nil {
		log.Fatal("初始化 kafka 失败", err)
	}
	return client
}
