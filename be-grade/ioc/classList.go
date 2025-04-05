package ioc

import (
	"context"
	classlistv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/classlist/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/spf13/viper"
	etcdv3 "go.etcd.io/etcd/client/v3"
	"time"
)

func InitClasslistClient(etcdClient *etcdv3.Client) classlistv1.ClasserClient {
	type Config struct {
		Endpoint string `yaml:"endpoint"`
		RetryCnt int    `yaml:"retryCnt"` //重连次数
	}
	var cfg Config
	//获取注册中心里面服务的名字
	err := viper.UnmarshalKey("grpc.client.classlist", &cfg)
	if err != nil {
		panic(err)
	}

	r := etcd.New(etcdClient)
	cc, err := grpc.DialInsecure(context.Background(),
		grpc.WithEndpoint(cfg.Endpoint),
		grpc.WithDiscovery(r),
		grpc.WithTimeout(5*time.Second), //5秒后自动超时
	)
	if err != nil {
		panic(err)
	}

	classlistClient := classlistv1.NewClasserClient(cc)
	return classlistClient
}
