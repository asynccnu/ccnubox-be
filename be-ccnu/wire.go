//go:build wireinject

package main

import (
	"github.com/asynccnu/ccnubox-be/be-ccnu/grpc"
	"github.com/asynccnu/ccnubox-be/be-ccnu/ioc"
	"github.com/asynccnu/ccnubox-be/be-ccnu/pkg/grpcx"
	"github.com/asynccnu/ccnubox-be/be-ccnu/service"
	"github.com/google/wire"
)

func InitGRPCServer() grpcx.Server {
	wire.Build(
		ioc.InitGRPCxKratosServer,
		grpc.NewCCNUServiceServer,
		service.NewCCNUService,
		ioc.InitLogger,
		ioc.InitEtcdClient,
	)
	return grpcx.Server(nil)
}
