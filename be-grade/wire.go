//go:generate wire
//go:build wireinject

package main

import (
	"github.com/asynccnu/ccnubox-be/be-grade/cron"
	"github.com/asynccnu/ccnubox-be/be-grade/grpc"
	"github.com/asynccnu/ccnubox-be/be-grade/ioc"
	"github.com/asynccnu/ccnubox-be/be-grade/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-grade/service"
	"github.com/google/wire"
)

func InitApp() App {
	wire.Build(
		grpc.NewGradeGrpcService,
		service.NewGradeService,
		service.NewRankService,
		dao.NewGradeDAO,
		dao.NewRankDAO,
		// 第三方
		ioc.InitEtcdClient,
		ioc.InitDB,
		ioc.InitLogger,
		ioc.InitGRPCxKratosServer,
		ioc.InitUserClient,
		ioc.InitCounterClient,
		ioc.InitFeedClient,
		ioc.InitClasslistClient,
		ioc.InitRedis,
		ioc.InitRedisLock,
		cron.NewGradeController,
		cron.NewCron,
		NewApp,
	)
	return App{}
}
