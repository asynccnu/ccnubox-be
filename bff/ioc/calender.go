package ioc

import (
	"context"

	"github.com/asynccnu/ccnubox-be/bff/conf"

	calendarv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/calendar/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitCalendarClient(ecli *clientv3.Client, cfg *conf.TransConf) calendarv1.CalendarServiceClient {
	const cal = "calendar"
	r := etcd.New(ecli)
	// grpc 通信
	cc, err := grpc.DialInsecure(context.Background(),
		grpc.WithEndpoint(cfg.Grpc.Client[cal].Endpoint),
		grpc.WithDiscovery(r),
	)
	if err != nil {
		panic(err)
	}

	// 初始化 calendar 的客户端
	client := calendarv1.NewCalendarServiceClient(cc)
	return client
}
