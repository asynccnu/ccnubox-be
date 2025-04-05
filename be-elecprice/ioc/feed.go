package ioc

import (
	"context"
	feedv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/feed/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/spf13/viper"
	etcdv3 "go.etcd.io/etcd/client/v3"
	"time"
)

func InitFeedClient(etcdClient *etcdv3.Client) feedv1.FeedServiceClient {
	type Config struct {
		Endpoint string `yaml:"endpoint"`
		RetryCnt int    `yaml:"retryCnt"` //重连次数
	}
	var cfg Config
	//获取注册中心里面服务的名字
	err := viper.UnmarshalKey("grpc.client.feed", &cfg)
	if err != nil {
		panic(err)
	}

	r := etcd.New(etcdClient)
	cc, err := grpc.DialInsecure(context.Background(),
		grpc.WithEndpoint(cfg.Endpoint),
		grpc.WithDiscovery(r),
		grpc.WithTimeout(10*time.Second), // TODO
	)
	if err != nil {
		panic(err)
	}

	client := feedv1.NewFeedServiceClient(cc)
	return client
}
