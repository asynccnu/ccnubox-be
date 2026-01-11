package ioc

import (
	"context"

	"github.com/asynccnu/ccnubox-be/bff/conf"
	bannerv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/banner/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitBannerClient(ecli *clientv3.Client, cfg *conf.TransConf) bannerv1.BannerServiceClient {
	const b = "banner"
	r := etcd.New(ecli)
	// grpc 通信
	cc, err := grpc.DialInsecure(context.Background(),
		grpc.WithEndpoint(cfg.Grpc.Client[b].Endpoint),
		grpc.WithDiscovery(r),
	)
	if err != nil {
		panic(err)
	}
	// 初始化 banner 的客户端
	client := bannerv1.NewBannerServiceClient(cc)
	return client
}
