//go:build wireinject

package main

import (
	"github.com/asynccnu/ccnubox-be/be-static/grpc"
	"github.com/asynccnu/ccnubox-be/be-static/ioc"
	"github.com/asynccnu/ccnubox-be/be-static/pkg/grpcx"
	"github.com/asynccnu/ccnubox-be/be-static/repository"
	"github.com/asynccnu/ccnubox-be/be-static/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-static/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-static/service"
	"github.com/google/wire"
)

func InitGRPCServer() grpcx.Server {
	wire.Build(
		ioc.InitGRPCxKratosServer,
		grpc.NewStaticServiceServer,
		service.NewStaticService,
		repository.NewCachedStaticRepository,
		cache.NewRedisStaticCache,
		dao.NewMongoDBStaticDAO,
		// 第三方
		ioc.InitDB,
		ioc.InitRedis,
		ioc.InitLogger,
		ioc.InitEtcdClient,
	)
	return grpcx.Server(nil)
}
