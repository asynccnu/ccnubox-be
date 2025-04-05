//go:build wireinject

package main

import (
	"github.com/asynccnu/ccnubox-be/be-user/grpc"
	"github.com/asynccnu/ccnubox-be/be-user/ioc"
	"github.com/asynccnu/ccnubox-be/be-user/pkg/grpcx"
	"github.com/asynccnu/ccnubox-be/be-user/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-user/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-user/service"
	"github.com/google/wire"
)

func InitGRPCServer() grpcx.Server {
	wire.Build(
		ioc.InitGRPCxKratosServer,
		grpc.NewUserServiceServer,
		service.NewUserService,
		dao.NewGORMUserDAO,
		cache.NewRedisUserCache,
		// 第三方
		ioc.InitCCNUClient,
		ioc.InitEtcdClient,
		ioc.NewCrypto,
		ioc.InitRedis,
		ioc.InitDB,
		ioc.InitLogger,
	)
	return grpcx.Server(nil)
}
