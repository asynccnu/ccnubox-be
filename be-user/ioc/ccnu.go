package ioc

import (
	"context"
	"time"

	"github.com/asynccnu/ccnubox-be/be-user/conf"
	ccnuv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/ccnu/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	etcdv3 "go.etcd.io/etcd/client/v3"
)

func InitCCNUClient(etcdClient *etcdv3.Client, cfg *conf.TransConf) ccnuv1.CCNUServiceClient {
	const c = "ccnu"

	r := etcd.New(etcdClient)
	cc, err := grpc.DialInsecure(context.Background(),
		grpc.WithEndpoint(cfg.Grpc.Client[c].Endpoint),
		grpc.WithDiscovery(r),
		grpc.WithTimeout(2*time.Minute), // 华师的超时设置为2分钟
		grpc.WithMiddleware(
			tracing.Client(),
		),
	)
	if err != nil {
		panic(err)
	}

	ccnuClient := ccnuv1.NewCCNUServiceClient(cc)
	return ccnuClient
}
