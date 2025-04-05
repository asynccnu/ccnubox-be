//go:generate wire
//go:build wireinject
// +build wireinject

package main

import (
	"github.com/asynccnu/ccnubox-be/be-calendar/cron"
	"github.com/asynccnu/ccnubox-be/be-calendar/grpc"
	"github.com/asynccnu/ccnubox-be/be-calendar/ioc"
	"github.com/asynccnu/ccnubox-be/be-calendar/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-calendar/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-calendar/service"
	"github.com/google/wire"
)

func InitApp() App {
	wire.Build(
		// 第三方
		ioc.InitDB,
		ioc.InitRedis,
		ioc.InitLogger,
		ioc.InitEtcdClient,
		ioc.InitFeedClient,
		ioc.InitQiniu,
		ioc.InitGRPCxKratosServer,
		grpc.NewCalendarServiceServer,
		service.NewCachedCalendarService,
		cache.NewRedisCalendarCache,
		dao.NewMysqlCalendarDAO,
		cron.NewHolidayController,
		cron.NewCalendarController,
		cron.NewCron,
		NewApp,
	)
	return App{}
}
