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
		// 跨域中间件(前置处理)
		corsMiddleware.MiddlewareFunc(), //在最开始就会请求达到的中间件,如果不符合跨域规范会被直接强制返回

		// 打点中间件(前后均处理)
		prometheusMiddleware.MiddlewareFunc(), // 需要在所有包含业务逻辑的中间件开始之前记录最完整的耗时;最后记录完整的耗时

		// 链路追踪注入中间件(前后均处理)
		otelMiddleware.Middleware(), // 请求开始的时候就会生成链路 id到上下文; 结束的时候会上报整个链路的所有信息

		// 日志中间件(后置处理)
		loggerMiddleware.MiddlewareFunc(), // 对于日志中间件需要等待所有的下游逻辑都结束之后再记录,因为这里除了记录日志同时还集中处理了响应,否则会出现某些上下文丢失情况

		// 链路追踪补充信息中间件(后置处理)
		otelMiddleware.AttributeMiddleware(), // 请求处理完成后生成相关信息,包括添加trace_id到Header,添加学号到链路等

	)

	// 创建用户认证中间件
	authMiddleware := loginMiddleware.MiddlewareFunc()

	// 注册路由
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
