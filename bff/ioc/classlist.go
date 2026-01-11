package ioc

import (
	"context"
	"time"

	"github.com/asynccnu/ccnubox-be/bff/conf"

	classlistv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/classlist/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitClassList(ecli *clientv3.Client, cfg *conf.TransConf) classlistv1.ClasserClient {
	const classls = "classlist"
	r := etcd.New(ecli)
	//grpc通信
	cc, err := grpc.DialInsecure(context.Background(),
		grpc.WithEndpoint(cfg.Grpc.Client[classls].Endpoint),
		grpc.WithDiscovery(r),
		grpc.WithTimeout(120*time.Second), //由于华师的速度比较慢这里地方需要强制给一个上下文超时的时间限制.否则kratos会使用默认的2s超时(有够脑瘫,为什么不自动沿用传入的ctx的上下文呢?)

	)
	if err != nil {
		panic(err)
	}
	//初始化static的客户端
	client := classlistv1.NewClasserClient(cc)
	return client
}
