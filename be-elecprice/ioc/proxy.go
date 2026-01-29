package ioc

import (
	"github.com/asynccnu/ccnubox-be/be-elecprice/conf"
	proxyv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/proxy/v1"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/grpc/client"
	etcdv3 "go.etcd.io/etcd/client/v3"
)

func InitProxyClient(etcdClient *etcdv3.Client, cfg *conf.InfraConf) proxyv1.ProxyClient {
	return client.InitProxy(etcdClient, cfg.Grpc, cfg.Env)
}
