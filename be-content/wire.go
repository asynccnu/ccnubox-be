//go:generate wire
//go:build wireinject
// +build wireinject

package main

import (
	"github.com/asynccnu/ccnubox-be/be-content/cron"
	"github.com/asynccnu/ccnubox-be/be-content/grpc"
	"github.com/asynccnu/ccnubox-be/be-content/ioc"
	"github.com/asynccnu/ccnubox-be/be-content/repository"
	"github.com/asynccnu/ccnubox-be/be-content/service"
	"github.com/google/wire"
)

func InitApp() App {
	wire.Build(
		// 第三方
		ioc.InitDB,
		ioc.InitRedis,
		ioc.InitLogger,
		ioc.InitEtcdClient,
		ioc.InitQiniu,
		ioc.InitGRPCxKratosServer,
		grpc.NewCalendarServiceServer,
		repository.ProviderSet,
		service.NewCalendarService,
		service.NewBannerService,
		service.NewDepartmentService,
		service.NewWebsiteService,
		service.NewInfoSumService,
		cron.NewCalendarController,
		cron.NewCron,
		NewApp,
	)
	return App{}
}
