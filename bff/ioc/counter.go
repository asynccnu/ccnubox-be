package ioc

import (
	"github.com/asynccnu/ccnubox-be/bff/conf"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/grpc/client"

	counterv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/counter/v1"
	etcdv3 "go.etcd.io/etcd/client/v3"
)

func InitCounterClient(ecli *etcdv3.Client, cfg *conf.InfraConf) counterv1.CounterServiceClient {
	return client.InitCounter(ecli, cfg.Grpc, cfg.Env)

}
