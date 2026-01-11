package ioc

import (
	"context"

	"github.com/asynccnu/ccnubox-be/bff/conf"
	websitev1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/website/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitWebsiteClient(ecli *clientv3.Client, cfg *conf.TransConf) websitev1.WebsiteServiceClient {
	const w = "website"
	r := etcd.New(ecli)
	// grpc 通信
	cc, err := grpc.DialInsecure(context.Background(),
		grpc.WithEndpoint(cfg.Grpc.Client[w].Endpoint),
		grpc.WithDiscovery(r),
	)
	if err != nil {
		panic(err)
	}
	// 初始化 website 的客户端
	client := websitev1.NewWebsiteServiceClient(cc)
	return client
}
