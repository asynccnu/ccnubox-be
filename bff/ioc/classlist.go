package ioc

import (
	"github.com/asynccnu/ccnubox-be/bff/conf"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/grpc/client"

	classlistv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/classlist/v1"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitClassListClient(ecli *clientv3.Client, cfg *conf.InfraConf) classlistv1.ClasserClient {
	return client.InitClassList(ecli, cfg.Grpc, cfg.Env)

}
