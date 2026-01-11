package ioc

import (
	"context"

	"github.com/asynccnu/ccnubox-be/bff/conf"
	infoSumv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/infoSum/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitInfoSumClient(ecli *clientv3.Client, cfg *conf.TransConf) infoSumv1.InfoSumServiceClient {
	const i = "infoSum"
	r := etcd.New(ecli)
	// grpc 通信
	cc, err := grpc.DialInsecure(context.Background(),
		grpc.WithEndpoint(cfg.Grpc.Client[i].Endpoint),
		grpc.WithDiscovery(r),
	)
	if err != nil {
		panic(err)
	}
	// 初始化 InfoSum 的客户端
	client := infoSumv1.NewInfoSumServiceClient(cc)
	return client
}
