package ioc

import (
	"github.com/asynccnu/ccnubox-be/bff/conf"
	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feed/v1"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/grpc/client"

	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitFeedClient(ecli *clientv3.Client, cfg *conf.InfraConf) feedv1.FeedServiceClient {
	return client.InitFeed(ecli, cfg.Grpc, cfg.Env)

}
