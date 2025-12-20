package ioc

import (
	"context"
	"time"

	proxyv1 "github.com/asynccnu/ccnubox-be/common/be-api/gen/proto/proxy/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/spf13/viper"
	etcdv3 "go.etcd.io/etcd/client/v3"
)

func InitProxyClient(etcdClient *etcdv3.Client) proxyv1.ProxyClient {
	type Config struct {
		Endpoint string `yaml:"endpoint"`
	}
	var cfg Config
	//获取注册中心里面服务的名字
	err := viper.UnmarshalKey("grpc.client.proxy", &cfg)
	if err != nil {
		panic(err)
	}

	r := etcd.New(etcdClient)
	cc, err := grpc.DialInsecure(context.Background(),
		grpc.WithEndpoint(cfg.Endpoint),
		grpc.WithDiscovery(r),
		grpc.WithTimeout(2*time.Minute), // 华师的超时设置为2分钟
	)
	if err != nil {
		panic(err)
	}

	ccnuClient := proxyv1.NewProxyClient(cc)
	return ccnuClient
}
