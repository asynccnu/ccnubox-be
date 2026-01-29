package server

import (
	"time"

	"github.com/asynccnu/ccnubox-be/common/bizpkg/conf"
	b_grpc "github.com/asynccnu/ccnubox-be/common/bizpkg/grpc"
	"github.com/asynccnu/ccnubox-be/common/pkg/grpcx"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	kgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

type GrpcServer interface {
	Register(server grpc.ServiceRegistrar)
}

func InitGRPCxKratosServer(
	grpcServer GrpcServer,
	ecli *clientv3.Client,
	l logger.Logger,
	cfg *conf.GrpcConf,
	env *conf.Env,
	middlewares ...middleware.Middleware,
) grpcx.Server {
	newCfg := *cfg
	// 添加前缀
	newCfg.Name = b_grpc.GetNamePrefix(env, newCfg.Name)
	s := kgrpc.NewServer(
		kgrpc.Address(cfg.Addr),
		kgrpc.Middleware(
			append([]middleware.Middleware{
				recovery.Recovery(),
				tracing.Server(),
				LoggingMiddleware(l),
			}, middlewares...)...,
		),
		kgrpc.Timeout(time.Duration(cfg.ServerTimeout)*time.Minute),
	)

	grpcServer.Register(s)
	return &grpcx.KratosServer{
		Server:     s,
		Name:       newCfg.Name,
		Weight:     newCfg.Weight,
		EtcdTTL:    time.Second * time.Duration(newCfg.EtcdTTL),
		EtcdClient: ecli,
		L:          l,
	}
}
