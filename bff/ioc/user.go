package ioc

import (
	"github.com/asynccnu/ccnubox-be/bff/conf"
	userv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/user/v1"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/grpc/client"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitUserClient(ecli *clientv3.Client, cfg *conf.InfraConf) userv1.UserServiceClient {
	return client.InitUser(ecli, cfg.Grpc, cfg.Env)
}
