package ioc

import (
	"context"
	websitev1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/website/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitWebsiteClient(ecli *clientv3.Client) websitev1.WebsiteServiceClient {
	// 配置 etcd 的路由
	type Config struct {
		Endpoint string `yaml:"endpoint"`
	}
	var cfg Config
	// 解析配置文件获取 website 的位置
	err := viper.UnmarshalKey("grpc.client.website", &cfg)
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
	// 初始化 website 的客户端
	client := websitev1.NewWebsiteServiceClient(cc)
	return client
}
