package ioc

import (
	"github.com/asynccnu/ccnubox-be/be-grade/conf"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/grpc/client"

	counterv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/counter/v1"
	etcdv3 "go.etcd.io/etcd/client/v3"
)

func InitCounterClient(etcdClient *etcdv3.Client, cfg *conf.InfraConf) counterv1.CounterServiceClient {
	return client.InitCounter(etcdClient, cfg.Grpc, cfg.Env)
}
