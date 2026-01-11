package ioc

import (
	"context"

	"github.com/asynccnu/ccnubox-be/bff/conf"

	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feedback_help/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitFeedbackHelpClient(ecli *clientv3.Client, cfg *conf.TransConf) feedv1.FeedbackHelpClient {
	const f = "feedbackHelp"
	r := etcd.New(ecli)
	// grpc 通信
	cc, err := grpc.DialInsecure(context.Background(),
		grpc.WithEndpoint(cfg.Grpc.Client[f].Endpoint),
		grpc.WithDiscovery(r),
	)
	if err != nil {
		panic(err)
	}
	// 初始化 feed 的客户端
	client := feedv1.NewFeedbackHelpClient(cc)
	return client
}
