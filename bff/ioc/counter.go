package ioc

import (
	"context"
	"time"

	"github.com/asynccnu/ccnubox-be/bff/conf"

	counterv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/counter/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	etcdv3 "go.etcd.io/etcd/client/v3"
)

func InitCounterClient(etcdClient *etcdv3.Client, cfg *conf.TransConf) counterv1.CounterServiceClient {
	const count = "counter"
	r := etcd.New(etcdClient)
	cc, err := grpc.DialInsecure(context.Background(),
		grpc.WithEndpoint(cfg.Grpc.Client[count].Endpoint),
		grpc.WithDiscovery(r),
		grpc.WithTimeout(10*time.Second), // TODO
	)
	if err != nil {
		panic(err)
	}

	feedUserCountClient := counterv1.NewCounterServiceClient(cc)
	return feedUserCountClient
}
