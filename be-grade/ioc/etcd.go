package ioc

import (
	"github.com/asynccnu/ccnubox-be/be-grade/conf"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/infra"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitEtcdClient(cfg *conf.InfraConf) *clientv3.Client {
	return infra.InitEtcdClient(cfg.Etcd)
}
