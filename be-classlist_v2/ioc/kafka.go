package ioc

import (
	"github.com/IBM/sarama"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/conf"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/infra"
)

func InitKafka(cfg *conf.InfraConf) sarama.Client {
	return infra.InitKafka(cfg.Kafka)
}
