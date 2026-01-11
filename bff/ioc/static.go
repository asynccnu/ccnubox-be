package ioc

import (
	"context"

	"github.com/asynccnu/ccnubox-be/bff/conf"
	staticv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/static/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitStaticClient(ecli *clientv3.Client, cfg *conf.TransConf) staticv1.StaticServiceClient {
	const s = "static"
	r := etcd.New(ecli)
	cc, err := grpc.DialInsecure(context.Background(),
		grpc.WithEndpoint(cfg.Grpc.Client[s].Endpoint),
		grpc.WithDiscovery(r),
	)
	if err != nil {
		panic(err)
	}
	client := staticv1.NewStaticServiceClient(cc)
	return client
}
