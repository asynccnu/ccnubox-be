package ioc

import (
	"github.com/asynccnu/ccnubox-be/be-grade/conf"
	userv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/user/v1"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/grpc/client"
	etcdv3 "go.etcd.io/etcd/client/v3"
)

func InitUserClient(etcdClient *etcdv3.Client, cfg *conf.InfraConf) userv1.UserServiceClient {
	return client.InitUser(etcdClient, cfg.Grpc, cfg.Env)
}
