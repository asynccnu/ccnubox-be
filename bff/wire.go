//go:generate wire
//go:build wireinject

package main

import (
	"github.com/asynccnu/ccnubox-be/bff/ioc"
	"github.com/asynccnu/ccnubox-be/bff/web/middleware"
	"github.com/google/wire"
)

func InitApp() *App {
	wire.Build(
		// 组件
		ioc.InitPrometheus,
		ioc.InitEtcdClient,
		ioc.InitLogger,
		ioc.InitRedis,
		//grpc注册
		ioc.InitDepartmentClient,
		ioc.InitWebsiteClient,
		ioc.InitBannerClient,
		ioc.InitCalendarClient,
		ioc.InitStaticClient,
		ioc.InitFeedClient,
		ioc.InitJwtHandler,
		ioc.InitUserClient,
		ioc.InitElecpriceClient,
		ioc.InitFeedbackHelpClient,
		ioc.InitGradeClient,
		ioc.InitInfoSumClient,
		ioc.InitCardClient,
		ioc.InitCounterClient,
		//基于kratos的微服务
		ioc.InitClassList,
		ioc.InitClassService,
		ioc.InitFreeClassroomClient,
		ioc.InitLibrary,

		//http服务
		ioc.InitPutPolicy,
		ioc.InitMac,
		ioc.InitClassRoomHandler,
		ioc.InitTubeHandler,
		ioc.InitUserHandler,
		ioc.InitBannerHandler,
		ioc.InitDepartmentHandler,
		ioc.InitCalendarHandler,
		ioc.InitWebsiteHandler,
		ioc.InitStaticHandler,
		ioc.InitFeedHandler,
		ioc.InitElecpriceHandler,
		ioc.InitClassHandler,
		ioc.InitGradeHandler,
		ioc.InitFeedbackHelpHandler,
		ioc.InitInfoSumHandler,
		ioc.InitCardHandler,
		ioc.InitMetricsHandel,
		ioc.InitLibraryHandler,

		//中间件
		middleware.NewLoggerMiddleware,
		middleware.NewCorsMiddleware,
		middleware.NewLoginMiddleWare,
		middleware.NewPrometheusMiddleware,
		//注册api
		ioc.InitGinServer,
		NewApp,
	)
	return &App{}
}
