//go:build wireinject

package main

import (
	"github.com/asynccnu/ccnubox-be/be-proxy/conf"
	"github.com/asynccnu/ccnubox-be/be-proxy/grpc"
	"github.com/asynccnu/ccnubox-be/be-proxy/ioc"
	"github.com/asynccnu/ccnubox-be/be-proxy/service"
	"github.com/google/wire"
)

func InitApp() *App {
	wire.Build(
		conf.InitInfraConfig,
		conf.InitServerConf,
		ioc.Provider,
		service.Provider,
		grpc.Provider,
		NewApp,
	)
	return &App{}
}
