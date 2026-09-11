//go:generate wire
//go:build wireinject
// +build wireinject

package main

import (
	"github.com/asynccnu/ccnubox-be/be-library/conf"
	"github.com/asynccnu/ccnubox-be/be-library/crawler"
	"github.com/asynccnu/ccnubox-be/be-library/grpc"
	"github.com/asynccnu/ccnubox-be/be-library/ioc"
	"github.com/asynccnu/ccnubox-be/be-library/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-library/service"
	"github.com/google/wire"
)

func InitApp() App {
	wire.Build(
		conf.InitInfraConfig,
		conf.InitServerConf,
		crawler.InitCrawlerHttpClient,
		grpc.NewLibraryGrpcService,
		service.NewSeatService,
		service.NewDiscussionService,
		service.NewCommentService,
		dao.NewCommentDAO,
		dao.NewReminderDAO,
		crawler.NewLibraryCrawlerMust,
		ioc.InitReminderCrawler,
		ioc.InitReminderFeedClient,
		ioc.InitReminderFeedGateway,
		service.NewReminderService,
		service.NewReminderScheduler,
		// 第三方
		ioc.InitEtcdClient,
		ioc.InitDB,
		ioc.InitLogger,
		ioc.InitMetrics,
		ioc.InitMetricsServer,
		ioc.InitGRPCxKratosServer,
		ioc.InitUserClient,
		ioc.InitProxyClient,
		ioc.InitHttpProxyClient,
		ioc.InitSecret,
		ioc.InitOTel,
		ioc.InitShutdown,
		NewApp,
	)
	return App{}
}
