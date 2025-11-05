//go:build wireinject

package main

import (
	"github.com/asynccnu/ccnubox-be/be-proxy/grpc"
	"github.com/asynccnu/ccnubox-be/be-proxy/ioc"
	"github.com/asynccnu/ccnubox-be/be-proxy/pkg/grpcx"
	"github.com/asynccnu/ccnubox-be/be-proxy/service"
	"github.com/google/wire"
)

func InitGRPCServer() grpcx.Server {
	wire.Build(
		ioc.Provider,
		service.Provider,
		grpc.Provider,
	)
	return grpcx.Server(nil)
}
