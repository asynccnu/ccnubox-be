package ioc

import (
	"context"
	ccnuv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/ccnu/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/spf13/viper"
	etcdv3 "go.etcd.io/etcd/client/v3"
	"time"
)

func InitCCNUClient(etcdClient *etcdv3.Client) ccnuv1.CCNUServiceClient {
	type Config struct {
		Endpoint string `yaml:"endpoint"`
		RetryCnt int    `yaml:"retryCnt"` //重连次数
	}
	var cfg Config
	//获取注册中心里面服务的名字
	err := viper.UnmarshalKey("grpc.client.ccnu", &cfg)
	if err != nil {
		panic(err)
	}

	r := etcd.New(etcdClient)
	cc, err := grpc.DialInsecure(context.Background(),
		grpc.WithEndpoint(cfg.Endpoint),
		grpc.WithDiscovery(r),
		grpc.WithTimeout(15*time.Second), // 华师的超时设置为15s
	)
	if err != nil {
		panic(err)
	}

	ccnuClient := ccnuv1.NewCCNUServiceClient(cc)
	return ccnuClient
}
