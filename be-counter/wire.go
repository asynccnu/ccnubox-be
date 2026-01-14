//go:build wireinject

package main

import (
	"github.com/asynccnu/ccnubox-be/be-counter/conf"
	"github.com/asynccnu/ccnubox-be/be-counter/grpc"
	"github.com/asynccnu/ccnubox-be/be-counter/ioc"
	"github.com/asynccnu/ccnubox-be/be-counter/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-counter/service"
	"github.com/google/wire"
)

func InitApp() *App {
	wire.Build(
		conf.InitInfraConfig,
		conf.InitServerConf,
		ioc.InitGRPCxKratosServer,
		grpc.NewCounterServiceServer,
		service.NewCachedCounterService,
		cache.NewRedisCounterCache,
		ioc.InitOTel,
		ioc.InitRedis,
		ioc.InitLogger,
		ioc.InitEtcdClient,
		NewApp,
	)
	return &App{}
}
