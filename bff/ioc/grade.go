package ioc

import (
	"context"
	"time"

	gradev1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/grade/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitGradeClient(ecli *clientv3.Client) gradev1.GradeServiceClient {
	//配置etcd的路由
	type Config struct {
		Endpoint string `yaml:"endpoint"`
	}
	var cfg Config

	//解析配置配置文件获取grade的位置
	err := viper.UnmarshalKey("grpc.client.grade", &cfg)
	if err != nil {
		panic(err)
	}

	r := etcd.New(ecli)
	//grpc通信
	cc, err := grpc.DialInsecure(context.Background(),
		grpc.WithEndpoint(cfg.Endpoint),
		grpc.WithDiscovery(r),
		// 由于排名查询太久了，把时间改久一点，有需要在调用时再将上下文超时改回30秒吧
		grpc.WithTimeout(3*time.Minute), //由于华师的速度比较慢这里地方需要强制给一个上下文超时的时间限制.否则kratos会使用默认的2s超时
	)
	if err != nil {
		panic(err)
	}
	//初始化static的客户端
	client := gradev1.NewGradeServiceClient(cc)
	return client
}
