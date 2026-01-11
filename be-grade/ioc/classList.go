package ioc

import (
	"context"
	"time"

	"github.com/asynccnu/ccnubox-be/be-grade/conf"
	classlistv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/classlist/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	etcdv3 "go.etcd.io/etcd/client/v3"
)

func InitClasslistClient(etcdClient *etcdv3.Client, cfg *conf.TransConf) classlistv1.ClasserClient {
	const cl = "classlist"
	r := etcd.New(etcdClient)
	cc, err := grpc.DialInsecure(context.Background(),
		grpc.WithEndpoint(cfg.Grpc.Client[cl].Endpoint),
		grpc.WithDiscovery(r),
		grpc.WithTimeout(5*time.Second), //5秒后自动超时
	)
	if err != nil {
		panic(err)
	}

	classlistClient := classlistv1.NewClasserClient(cc)
	return classlistClient
}
