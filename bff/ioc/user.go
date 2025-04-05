package ioc

import (
	"context"
	userv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/user/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitUserClient(ecli *clientv3.Client) userv1.UserServiceClient {
	//初始化UserClient用于和下游的用户服务交互,可以看到这里注入了etcd
	type Config struct {
		Endpoint string `yaml:"endpoint"` //etcd暴露的端口
	}
	var cfg Config

	err := viper.UnmarshalKey("grpc.client.user", &cfg)
	if err != nil {
		panic(err)
	}

	r := etcd.New(ecli)
	//grpc启动!
	cc, err := grpc.DialInsecure(context.Background(),
		grpc.WithEndpoint(cfg.Endpoint),
		grpc.WithDiscovery(r),
	)
	if err != nil {
		panic(err)
	}
	//创建一个用户服务实体
	client := userv1.NewUserServiceClient(cc)
	return client
}
