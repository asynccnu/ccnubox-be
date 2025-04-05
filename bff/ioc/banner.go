package ioc

import (
	"context"
	bannerv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/banner/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitBannerClient(ecli *clientv3.Client) bannerv1.BannerServiceClient {
	// 配置 etcd 的路由
	type Config struct {
		Endpoint string `yaml:"endpoint"`
	}
	var cfg Config
	// 解析配置文件获取 banner 的位置
	err := viper.UnmarshalKey("grpc.client.banner", &cfg)
	if err != nil {
		panic(err)
	}
	r := etcd.New(ecli)
	// grpc 通信
	cc, err := grpc.DialInsecure(context.Background(),
		grpc.WithEndpoint(cfg.Endpoint),
		grpc.WithDiscovery(r),
	)
	if err != nil {
		panic(err)
	}
	// 初始化 banner 的客户端
	client := bannerv1.NewBannerServiceClient(cc)
	return client
}
