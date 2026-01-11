package ioc

import (
	"context"

	"github.com/asynccnu/ccnubox-be/bff/conf"
	userv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/user/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/grpcx"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitUserClient(ecli *clientv3.Client, cfg *conf.TransConf) userv1.UserServiceClient {
	const u = "user"
	r := etcd.New(ecli)
	//grpc启动!
	cc := grpcx.Dial(context.Background(), r, cfg.Grpc.Client[u].Endpoint)

	//创建一个用户服务实体
	client := userv1.NewUserServiceClient(cc)
	return client
}
