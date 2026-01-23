//go:generate wire
//go:build wireinject

package main

import (
	"github.com/asynccnu/ccnubox-be/be-elecprice/conf"
	"github.com/asynccnu/ccnubox-be/be-elecprice/cron"
	"github.com/asynccnu/ccnubox-be/be-elecprice/grpc"
	"github.com/asynccnu/ccnubox-be/be-elecprice/ioc"
	"github.com/asynccnu/ccnubox-be/be-elecprice/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-elecprice/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-elecprice/service"
	"github.com/google/wire"
)

func InitApp() *App {
	wire.Build(
		conf.InitInfraConfig,
		conf.InitServerConf,
		grpc.NewElecpriceGrpcService,
		service.NewElecpriceService,
		dao.NewElecpriceDAO,
		cache.NewRedisElecPriceCache,
		// 第三方
		ioc.InitRedis,
		ioc.InitEtcdClient,
		ioc.InitProxyClient,
		ioc.InitDB,
		ioc.InitLogger,
		ioc.InitGRPCxKratosServer,
		ioc.InitFeedClient,
		ioc.InitOTel,
		cron.NewElecpriceController,
		cron.NewCron,
		NewApp,
	)
	return &App{}
}
