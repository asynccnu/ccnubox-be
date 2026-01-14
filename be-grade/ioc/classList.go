package ioc

import (
	"github.com/asynccnu/ccnubox-be/be-grade/conf"
	classlistv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/classlist/v1"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/grpc/client"
	etcdv3 "go.etcd.io/etcd/client/v3"
)

func InitClassListClient(etcdClient *etcdv3.Client, cfg *conf.InfraConf) classlistv1.ClasserClient {
	return client.InitClassList(etcdClient, cfg.Grpc, cfg.Env)
}
