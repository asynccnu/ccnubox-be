package ioc

import (
	"context"
	"time"

	"github.com/asynccnu/ccnubox-be/be-grade/conf"
	proxyv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/proxy/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	etcdv3 "go.etcd.io/etcd/client/v3"
)

func InitProxyClient(etcdClient *etcdv3.Client, cfg *conf.TransConf) proxyv1.ProxyClient {
	const p = "proxy"
	r := etcd.New(etcdClient)
	cc, err := grpc.DialInsecure(context.Background(),
		grpc.WithEndpoint(cfg.Grpc.Client[p].Endpoint),
		grpc.WithDiscovery(r),
		grpc.WithTimeout(2*time.Minute), // 华师的超时设置为2分钟
	)
	if err != nil {
		panic(err)
	}

	ccnuClient := proxyv1.NewProxyClient(cc)
	return ccnuClient
}
