//go:build wireinject

package main

import (
	"github.com/asynccnu/ccnubox-be/be-proxy/conf"
	"github.com/asynccnu/ccnubox-be/be-proxy/grpc"
	"github.com/asynccnu/ccnubox-be/be-proxy/ioc"
	"github.com/asynccnu/ccnubox-be/be-proxy/service"
	"github.com/asynccnu/ccnubox-be/common/pkg/grpcx"
	"github.com/google/wire"
)

func InitGRPCServer() grpcx.Server {
	wire.Build(
		conf.InitInfraConfig,
		conf.InitTransConfig,
		ioc.Provider,
		service.Provider,
		grpc.Provider,
	)
	return grpcx.Server(nil)
}
