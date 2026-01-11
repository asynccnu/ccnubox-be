package ioc

import (
	"context"
	"time"

	"github.com/asynccnu/ccnubox-be/be-grade/conf"
	userv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/user/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	etcdv3 "go.etcd.io/etcd/client/v3"
)

func InitUserClient(etcdClient *etcdv3.Client, cfg *conf.TransConf) userv1.UserServiceClient {
	const u = "user"
	r := etcd.New(etcdClient)
	cc, err := grpc.DialInsecure(context.Background(),
		grpc.WithEndpoint(cfg.Grpc.Client[u].Endpoint),
		grpc.WithDiscovery(r),
		grpc.WithTimeout(2*time.Minute), //涉及华师的服务都改成2分钟
	)
	if err != nil {
		panic(err)
	}

	userClient := userv1.NewUserServiceClient(cc)
	return userClient
}
