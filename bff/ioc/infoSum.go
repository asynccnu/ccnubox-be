package ioc

import (
	"context"
	infoSumv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/infoSum/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitInfoSumClient(ecli *clientv3.Client) infoSumv1.InfoSumServiceClient {
	// 配置 etcd 的路由
	type Config struct {
		Endpoint string `yaml:"endpoint"`
	}
	var cfg Config
	// 解析配置文件获取 InfoSum 的位置
	err := viper.UnmarshalKey("grpc.client.infoSum", &cfg)
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
	// 初始化 InfoSum 的客户端
	client := infoSumv1.NewInfoSumServiceClient(cc)
	return client
}
