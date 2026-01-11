package ioc

import (
	"context"
	"time"

	"github.com/asynccnu/ccnubox-be/bff/conf"

	cardv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/card/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitCardClient(ecli *clientv3.Client, cfg *conf.TransConf) cardv1.CardClient {
	const card = "card"
	r := etcd.New(ecli)
	// grpc 通信
	cc, err := grpc.DialInsecure(context.Background(),
		grpc.WithEndpoint(cfg.Grpc.Client[card].Endpoint),
		grpc.WithDiscovery(r),
		grpc.WithTimeout(10*time.Second), //这里给了华师10秒的超时连接设置
	)
	if err != nil {
		panic(err)
	}
	// 初始化 card 的客户端
	client := cardv1.NewCardClient(cc)
	return client
}
