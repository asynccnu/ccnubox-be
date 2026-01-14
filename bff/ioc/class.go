package ioc

import (
	"github.com/asynccnu/ccnubox-be/bff/conf"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/grpc/client"

	cs "github.com/asynccnu/ccnubox-be/common/api/gen/proto/classService/v1"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitClassClient(ecli *clientv3.Client, cfg *conf.InfraConf) cs.ClassServiceClient {
	return client.InitClass(ecli, cfg.Grpc, cfg.Env)
}
