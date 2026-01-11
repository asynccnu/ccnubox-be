package ioc

import (
	"context"

	contentv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/content/v1"

	"github.com/asynccnu/ccnubox-be/bff/conf"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitContentClient(ecli *clientv3.Client, cfg *conf.TransConf) contentv1.ContentServiceClient {
	const w = "content"
	r := etcd.New(ecli)
	// grpc 通信
	cc, err := grpc.DialInsecure(context.Background(),
		grpc.WithEndpoint(cfg.Grpc.Client[w].Endpoint),
		grpc.WithDiscovery(r),
	)
	if err != nil {
		panic(err)
	}
	// 初始化 content 的客户端
	client := contentv1.NewContentServiceClient(cc)
	return client
}
