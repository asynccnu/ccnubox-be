//go:generate wire
//go:build wireinject

package main

import (
	"github.com/asynccnu/ccnubox-be/bff/conf"
	"github.com/asynccnu/ccnubox-be/bff/ioc"
	"github.com/asynccnu/ccnubox-be/bff/web/middleware"
	"github.com/google/wire"
)

func InitApp() *App {
	wire.Build(
		conf.InitInfraConfig,
		conf.InitTransConfig,
		// 组件
		ioc.InitPrometheus,
		ioc.InitEtcdClient,
		ioc.InitLogger,
		ioc.InitRedis,
		//grpc注册
		ioc.InitFeedClient,
		ioc.InitJwtHandler,
		ioc.InitUserClient,
		ioc.InitElecpriceClient,
		ioc.InitGradeClient,
		ioc.InitContentClient,
		ioc.InitCounterClient,
		//基于kratos的微服务
		ioc.InitClassList,
		ioc.InitClassService,
		ioc.InitFreeClassroomClient,
		ioc.InitLibrary,

		//http服务
		ioc.InitTubePolicies,
		ioc.InitMac,
		ioc.InitClassRoomHandler,
		ioc.InitTubeHandler,
		ioc.InitUserHandler,
		ioc.InitContentHandler,
		ioc.InitFeedHandler,
		ioc.InitElecpriceHandler,
		ioc.InitClassHandler,
		ioc.InitGradeHandler,
		ioc.InitMetricsHandel,
		ioc.InitLibraryHandler,
		ioc.InitSwagHandler,

		//中间件
		middleware.NewLoggerMiddleware,
		middleware.NewCorsMiddleware,
		middleware.NewLoginMiddleWare,
		middleware.NewPrometheusMiddleware,
		middleware.NewBasicAuthMiddleware,
		middleware.NewOtelMiddlerware,
		//注册api
		ioc.InitGinServer,
		NewApp,
	)
	return &App{}
}
