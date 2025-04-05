package ioc

import (
	"context"
	elecpricev1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/elecprice/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitElecpriceClient(ecli *clientv3.Client) elecpricev1.ElecpriceServiceClient {
	//配置etcd的路由
	type Config struct {
		Endpoint string `yaml:"endpoint"`
	}
	var cfg Config

	//解析配置配置文件获取department的位置
	err := viper.UnmarshalKey("grpc.client.elecprice", &cfg)
	if err != nil {
		panic(err)
	}
	r := etcd.New(ecli)
	//grpc通信
	cc, err := grpc.DialInsecure(context.Background(),
		grpc.WithEndpoint(cfg.Endpoint),
		grpc.WithDiscovery(r),
	)
	if err != nil {
		panic(err)
	}
	//初始化static的客户端
	client := elecpricev1.NewElecpriceServiceClient(cc)
	return client
}
