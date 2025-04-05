package ioc

import (
	"context"
	classlistv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/classlist/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

func InitClassList(ecli *clientv3.Client) classlistv1.ClasserClient {
	//配置etcd的路由
	type Config struct {
		Endpoint string `yaml:"endpoint"`
	}
	var cfg Config

	//解析配置配置文件获取department的位置
	err := viper.UnmarshalKey("grpc.client.classlist", &cfg)
	if err != nil {
		panic(err)
	}
	r := etcd.New(ecli)
	//grpc通信
	cc, err := grpc.DialInsecure(context.Background(),
		grpc.WithEndpoint(cfg.Endpoint),
		grpc.WithDiscovery(r),
		grpc.WithTimeout(30*time.Second), //由于华师的速度比较慢这里地方需要强制给一个上下文超时的时间限制.否则kratos会使用默认的2s超时(有够脑瘫,为什么不自动沿用传入的ctx的上下文呢?)

	)
	if err != nil {
		panic(err)
	}
	//初始化static的客户端
	client := classlistv1.NewClasserClient(cc)
	return client
}
