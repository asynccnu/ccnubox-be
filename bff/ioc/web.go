package ioc

import (
	"github.com/asynccnu/ccnubox-be/bff/web/banner"
	"github.com/asynccnu/ccnubox-be/bff/web/calendar"
	"github.com/asynccnu/ccnubox-be/bff/web/card"
	"github.com/asynccnu/ccnubox-be/bff/web/class"
	"github.com/asynccnu/ccnubox-be/bff/web/classroom"
	"github.com/asynccnu/ccnubox-be/bff/web/department"
	"github.com/asynccnu/ccnubox-be/bff/web/elecprice"
	"github.com/asynccnu/ccnubox-be/bff/web/feed"
	"github.com/asynccnu/ccnubox-be/bff/web/feedback_help"
	"github.com/asynccnu/ccnubox-be/bff/web/grade"
	"github.com/asynccnu/ccnubox-be/bff/web/infoSum"
	"github.com/asynccnu/ccnubox-be/bff/web/library"
	"github.com/asynccnu/ccnubox-be/bff/web/metrics"
	"github.com/asynccnu/ccnubox-be/bff/web/middleware"
	"github.com/asynccnu/ccnubox-be/bff/web/static"
	"github.com/asynccnu/ccnubox-be/bff/web/swag"
	"github.com/asynccnu/ccnubox-be/bff/web/tube"
	"github.com/asynccnu/ccnubox-be/bff/web/user"
	"github.com/asynccnu/ccnubox-be/bff/web/website"
	"github.com/gin-gonic/gin"
)

func InitGinServer(
	loggerMiddleware *middleware.LoggerMiddleware,
	loginMiddleware *middleware.LoginMiddleware,
	corsMiddleware *middleware.CorsMiddleware,
	basicAuthMiddleware *middleware.BasicAuthMiddleware,
	prometheusMiddleware *middleware.PrometheusMiddleware,
	otelMiddleware *middleware.OtelMiddleware,
	classroom *classroom.ClassRoomHandler,
	tube *tube.TubeHandler,
	user *user.UserHandler,
	static *static.StaticHandler,
	banner *banner.BannerHandler,
	department *department.DepartmentHandler,
	website *website.WebsiteHandler,
	calendar *calendar.CalendarHandler,
	feed *feed.FeedHandler,
	elecprice *elecprice.ElecPriceHandler, //添加你的服务handler
	grade *grade.GradeHandler,
	class *class.ClassHandler,
	feedback *feedback_help.FeedbackHelpHandler,
	infoSum *infoSum.InfoSumHandler,
	card *card.CardHandler,
	metrics *metrics.MetricsHandler,
	library *library.LibraryHandler,
	swag *swag.SwagHandler,
) *gin.Engine {
	//初始化一个gin引擎
	engine := gin.Default()
	//全局使用gin中间件
	api := engine.Group("/api/v1")

	//使用中间件
	api.Use(
		// 跨域中间件
		corsMiddleware.MiddlewareFunc(),
		// 打点中间件
		prometheusMiddleware.MiddlewareFunc(),
		// 日志中间件
		loggerMiddleware.MiddlewareFunc(),
		// 追踪中间件
		otelMiddleware.Middleware(),
	)

	//创建用户认证中间件
	authMiddleware := loginMiddleware.MiddlewareFunc()

	//注册一堆路由
	user.RegisterRoutes(api, authMiddleware)
	static.RegisterRoutes(api, authMiddleware)
	banner.RegisterRoutes(api, authMiddleware)
	department.RegisterRoutes(api, authMiddleware)
	website.RegisterRoutes(api, authMiddleware)
	calendar.RegisterRoutes(api, authMiddleware)
	feed.RegisterRoutes(api, authMiddleware)
	elecprice.RegisterRoutes(api, authMiddleware)
	class.RegisterRoutes(api, authMiddleware)
	feedback.RegisterRoutes(api, authMiddleware)
	infoSum.RegisterRoutes(api, authMiddleware)
	grade.RegisterRoutes(api, authMiddleware)
	card.RegisterRoute(api, authMiddleware)
	tube.RegisterRoutes(api, authMiddleware, basicAuthMiddleware.MiddlewareFunc())
	metrics.RegisterRoutes(api, basicAuthMiddleware.MiddlewareFunc(), authMiddleware)
	classroom.RegisterRoutes(api, authMiddleware)
	library.RegisterRoutes(api, authMiddleware)
	swag.RegisterRoutes(api, basicAuthMiddleware.MiddlewareFunc())

	//返回路由
	return engine
}
