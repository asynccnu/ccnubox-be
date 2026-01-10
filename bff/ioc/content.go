package ioc

import (
	"context"
	contentv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/content/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitContentClient(ecli *clientv3.Client) contentv1.ContentServiceClient {
	// 配置 etcd 的路由
	type Config struct {
		Endpoint string `yaml:"endpoint"`
	}
	var cfg Config
	// 解析配置文件获取 content 的位置
	err := viper.UnmarshalKey("grpc.client.content", &cfg)
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
	// 初始化 content 的客户端
	client := contentv1.NewContentServiceClient(cc)
	return client
}
