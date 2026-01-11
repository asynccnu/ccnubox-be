//go:build wireinject
// +build wireinject

package main

import (
	"github.com/asynccnu/ccnubox-be/be-counter/conf"
	"github.com/asynccnu/ccnubox-be/be-counter/grpc"
	"github.com/asynccnu/ccnubox-be/be-counter/ioc"
	"github.com/asynccnu/ccnubox-be/be-counter/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-counter/service"
	"github.com/asynccnu/ccnubox-be/common/pkg/grpcx"
	"github.com/google/wire"
)

func InitGRPCServer() grpcx.Server {
	wire.Build(
		conf.InitInfraConfig,
		conf.InitTransConfig,
		ioc.InitGRPCxKratosServer,
		grpc.NewCounterServiceServer,
		service.NewCachedCounterService,
		cache.NewRedisCounterCache,
		// 第三方
		ioc.InitRedis,
		ioc.InitLogger,
		ioc.InitEtcdClient,
	)
	return grpcx.Server(nil)
}
