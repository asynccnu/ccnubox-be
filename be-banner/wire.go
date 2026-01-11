//go:build wireinject
// +build wireinject

package main

import (
	"github.com/asynccnu/ccnubox-be/be-banner/conf"
	"github.com/asynccnu/ccnubox-be/be-banner/grpc"
	"github.com/asynccnu/ccnubox-be/be-banner/ioc"
	"github.com/asynccnu/ccnubox-be/be-banner/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-banner/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-banner/service"
	"github.com/asynccnu/ccnubox-be/common/pkg/grpcx"
	"github.com/google/wire"
)

func InitGRPCServer() grpcx.Server {
	wire.Build(
		conf.InitInfraConfig,
		conf.InitTransConfig,
		ioc.InitGRPCxKratosServer,
		grpc.NewBannerServiceServer,
		service.NewBannerService,
		cache.NewRedisBannerCache,
		dao.NewMysqlBannerDAO,
		// 第三方
		ioc.InitDB,
		ioc.InitRedis,
		ioc.InitLogger,
		ioc.InitEtcdClient,
	)
	return grpcx.Server(nil)
}
