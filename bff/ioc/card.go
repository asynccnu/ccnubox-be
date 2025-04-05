package ioc

import (
	"context"
	cardv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/card/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

func InitCardClient(ecli *clientv3.Client) cardv1.CardClient {
	// 配置 etcd 的路由
	type Config struct {
		Endpoint string `yaml:"endpoint"`
	}
	var cfg Config
	// 解析配置文件获取 calendar 的位置
	err := viper.UnmarshalKey("grpc.client.card", &cfg)
	if err != nil {
		panic(err)
	}
	r := etcd.New(ecli)
	// grpc 通信
	cc, err := grpc.DialInsecure(context.Background(),
		grpc.WithEndpoint(cfg.Endpoint),
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
