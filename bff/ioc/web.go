package ioc

import (
	"github.com/asynccnu/ccnubox-be/bff/web/class"
	"github.com/asynccnu/ccnubox-be/bff/web/classroom"
	"github.com/asynccnu/ccnubox-be/bff/web/content"
	"github.com/asynccnu/ccnubox-be/bff/web/elecprice"
	"github.com/asynccnu/ccnubox-be/bff/web/feed"
	"github.com/asynccnu/ccnubox-be/bff/web/grade"
	"github.com/asynccnu/ccnubox-be/bff/web/library"
	"github.com/asynccnu/ccnubox-be/bff/web/metrics"
	"github.com/asynccnu/ccnubox-be/bff/web/middleware"
	"github.com/asynccnu/ccnubox-be/bff/web/swag"
	"github.com/asynccnu/ccnubox-be/bff/web/tube"
	"github.com/asynccnu/ccnubox-be/bff/web/user"
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
	feed *feed.FeedHandler,
	elecprice *elecprice.ElecPriceHandler,
	grade *grade.GradeHandler,
	class *class.ClassHandler,
	content *content.ContentHandler,
	metrics *metrics.MetricsHandler,
	library *library.LibraryHandler,
	swag *swag.SwagHandler,
) *gin.Engine {
	// 初始化一个gin引擎
	engine := gin.Default()

	// 开启 ginContext 自动回退机制
	// 当 gin.Context 找不到所指定的 key 时
	// 它会自动去 c.Request.Context() 里面找
	engine.ContextWithFallback = true

	// 全局使用gin中间件
	api := engine.Group("/api/v1")

	// 使用中间件
	api.Use(
		// 跨域中间件
		corsMiddleware.MiddlewareFunc(),
		// 追踪中间件
		otelMiddleware.Middleware(),
		otelMiddleware.AttributeMiddleware(),
		// 打点中间件
		prometheusMiddleware.MiddlewareFunc(),
		// 日志中间件
		loggerMiddleware.MiddlewareFunc(),
	)

	// 创建用户认证中间件
	authMiddleware := loginMiddleware.MiddlewareFunc()

	// 注册一堆路由
	user.RegisterRoutes(api, authMiddleware)
	content.RegisterRoutes(api, authMiddleware)
	feed.RegisterRoutes(api, authMiddleware, basicAuthMiddleware.MiddlewareFunc())
	elecprice.RegisterRoutes(api, authMiddleware)
	class.RegisterRoutes(api, authMiddleware)
	grade.RegisterRoutes(api, authMiddleware)
	tube.RegisterRoutes(api, authMiddleware, basicAuthMiddleware.MiddlewareFunc())
	metrics.RegisterRoutes(api, basicAuthMiddleware.MiddlewareFunc(), authMiddleware)
	classroom.RegisterRoutes(api, authMiddleware)
	library.RegisterRoutes(api, authMiddleware)
	swag.RegisterRoutes(api, basicAuthMiddleware.MiddlewareFunc())

	// 返回路由
	return engine
}
