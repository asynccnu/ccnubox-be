package ioc

import (
	"github.com/asynccnu/ccnubox-be/bff/conf"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/grpc/client"

	elecpricev1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/elecprice/v1"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitElecpriceClient(ecli *clientv3.Client, cfg *conf.InfraConf) elecpricev1.ElecpriceServiceClient {
	return client.InitElecprice(ecli, cfg.Grpc, cfg.Env)
}
