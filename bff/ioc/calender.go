package ioc

import (
	"context"
	calendarv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/calendar/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitCalendarClient(ecli *clientv3.Client) calendarv1.CalendarServiceClient {
	// 配置 etcd 的路由
	type Config struct {
		Endpoint string `yaml:"endpoint"`
	}
	var cfg Config
	// 解析配置文件获取 calendar 的位置
	err := viper.UnmarshalKey("grpc.client.calendar", &cfg)
	if err != nil {
		panic(err)
	}
	r := etcd.New(ecli)
	// grpc 通信
	cc, err := grpc.DialInsecure(context.Background(),
		grpc.WithEndpoint(cfg.Endpoint),
		grpc.WithDiscovery(r),
	)
	if err != nil {
		panic(err)
	}

	// 初始化 calendar 的客户端
	client := calendarv1.NewCalendarServiceClient(cc)
	return client
}
