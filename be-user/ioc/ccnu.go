package ioc

import (
	"github.com/asynccnu/ccnubox-be/be-user/conf"
	ccnuv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/ccnu/v1"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/grpc/client"
	etcdv3 "go.etcd.io/etcd/client/v3"
)

func InitCCNUClient(etcdClient *etcdv3.Client, cfg *conf.InfraConf) ccnuv1.CCNUServiceClient {
	return client.InitCCNU(etcdClient, cfg.Grpc, cfg.Env)
}
