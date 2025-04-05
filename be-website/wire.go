//go:build wireinject

package main

import (
	"github.com/asynccnu/ccnubox-be/be-website/grpc"
	"github.com/asynccnu/ccnubox-be/be-website/ioc"
	"github.com/asynccnu/ccnubox-be/be-website/pkg/grpcx"
	"github.com/asynccnu/ccnubox-be/be-website/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-website/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-website/service"
	"github.com/google/wire"
)

func InitGRPCServer() grpcx.Server {
	wire.Build(
		ioc.InitGRPCxKratosServer,
		grpc.NewWebsiteServiceServer,
		service.NewWebsiteService,
		cache.NewRedisWebsiteCache,
		dao.NewMysqlWebsiteDAO,
		// 第三方
		ioc.InitDB,
		ioc.InitRedis,
		ioc.InitLogger,
		ioc.InitEtcdClient,
	)
	return grpcx.Server(nil)
}
