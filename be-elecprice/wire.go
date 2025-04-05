//go:generate wire
//go:build wireinject

package main

import (
	"github.com/asynccnu/ccnubox-be/be-elecprice/cron"
	"github.com/asynccnu/ccnubox-be/be-elecprice/grpc"
	"github.com/asynccnu/ccnubox-be/be-elecprice/ioc"
	"github.com/asynccnu/ccnubox-be/be-elecprice/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-elecprice/service"
	"github.com/google/wire"
)

func InitApp() App {
	wire.Build(
		grpc.NewElecpriceGrpcService,
		service.NewElecpriceService,
		dao.NewElecpriceDAO,
		// 第三方
		ioc.InitEtcdClient,
		ioc.InitDB,
		ioc.InitLogger,
		ioc.InitGRPCxKratosServer,
		ioc.InitFeedClient,
		cron.NewElecpriceController,
		cron.NewCron,
		NewApp,
	)
	return App{}
}
