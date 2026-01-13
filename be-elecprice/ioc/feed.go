package ioc

import (
	"github.com/asynccnu/ccnubox-be/be-elecprice/conf"
	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feed/v1"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/grpc/client"
	etcdv3 "go.etcd.io/etcd/client/v3"
)

func InitFeedClient(etcdClient *etcdv3.Client, cfg *conf.InfraConf) feedv1.FeedServiceClient {
	return client.InitFeed(etcdClient, cfg.Grpc, cfg.Env)
}
