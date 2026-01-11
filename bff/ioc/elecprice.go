package ioc

import (
	"context"
	"time"

	"github.com/asynccnu/ccnubox-be/bff/conf"

	elecpricev1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/elecprice/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitElecpriceClient(ecli *clientv3.Client, cfg *conf.TransConf) elecpricev1.ElecpriceServiceClient {
	const e = "elecprice"
	r := etcd.New(ecli)
	//grpc通信
	cc, err := grpc.DialInsecure(context.Background(),
		grpc.WithEndpoint(cfg.Grpc.Client[e].Endpoint),
		grpc.WithDiscovery(r),
		grpc.WithTimeout(120*time.Second),
	)
	if err != nil {
		panic(err)
	}
	//初始化static的客户端
	client := elecpricev1.NewElecpriceServiceClient(cc)
	return client
}
