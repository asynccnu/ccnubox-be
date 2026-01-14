package ioc

import (
	contentv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/content/v1"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/grpc/client"

	"github.com/asynccnu/ccnubox-be/bff/conf"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitContentClient(ecli *clientv3.Client, cfg *conf.InfraConf) contentv1.ContentServiceClient {
	return client.InitContent(ecli, cfg.Grpc, cfg.Env)

}
