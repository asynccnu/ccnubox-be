//go:build wireinject

package main

import (
	"github.com/asynccnu/ccnubox-be/be-ccnu/conf"
	"github.com/asynccnu/ccnubox-be/be-ccnu/grpc"
	"github.com/asynccnu/ccnubox-be/be-ccnu/ioc"
	"github.com/asynccnu/ccnubox-be/be-ccnu/service"
	"github.com/google/wire"
)

func InitApp() *App {
	wire.Build(
		conf.InitInfraConfig,
		conf.InitServerConf,
		ioc.InitGRPCxKratosServer,
		grpc.NewCCNUServiceServer,
		service.NewCCNUService,
		ioc.InitOTel,
		ioc.InitProxyClient,
		ioc.InitLogger,
		ioc.InitEtcdClient,
		NewApp,
	)
	return &App{}
}
