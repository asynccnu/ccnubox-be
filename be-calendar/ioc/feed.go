package ioc

import (
	"context"
	"time"

	"github.com/asynccnu/ccnubox-be/be-calendar/conf"
	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feed/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	etcdv3 "go.etcd.io/etcd/client/v3"
)

func InitFeedClient(etcdClient *etcdv3.Client, cfg *conf.TransConf) feedv1.FeedServiceClient {
	const f = "feed"

	r := etcd.New(etcdClient)
	cc, err := grpc.DialInsecure(context.Background(),
		grpc.WithEndpoint(cfg.Grpc.Client[f].Endpoint),
		grpc.WithDiscovery(r),
		grpc.WithTimeout(10*time.Second), // TODO
	)
	if err != nil {
		panic(err)
	}

	client := feedv1.NewFeedServiceClient(cc)
	return client
}
