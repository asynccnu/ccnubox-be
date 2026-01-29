package ioc

import (
	"github.com/asynccnu/ccnubox-be/bff/conf"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/grpc/client"

	libraryv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/library/v1"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitLibrary(ecli *clientv3.Client, cfg *conf.InfraConf) libraryv1.LibraryClient {
	return client.InitLibrary(ecli, cfg.Grpc, cfg.Env)

}
