package ioc

import (
	"context"
	"github.com/asynccnu/ccnubox-be/bff/web/banner"
	"github.com/asynccnu/ccnubox-be/bff/web/calendar"
	"github.com/asynccnu/ccnubox-be/bff/web/card"
	"github.com/asynccnu/ccnubox-be/bff/web/class"
	"github.com/asynccnu/ccnubox-be/bff/web/department"
	"github.com/asynccnu/ccnubox-be/bff/web/elecprice"
	"github.com/asynccnu/ccnubox-be/bff/web/feed"
	"github.com/asynccnu/ccnubox-be/bff/web/feedback_help"
	"github.com/asynccnu/ccnubox-be/bff/web/grade"
	"github.com/asynccnu/ccnubox-be/bff/web/infoSum"
	"github.com/asynccnu/ccnubox-be/bff/web/metrics"
	"github.com/asynccnu/ccnubox-be/bff/web/middleware"
	"github.com/asynccnu/ccnubox-be/bff/web/static"
	"github.com/asynccnu/ccnubox-be/bff/web/tube"
	"github.com/asynccnu/ccnubox-be/bff/web/user"
	"github.com/asynccnu/ccnubox-be/bff/web/website"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"time"
)

// 逆天参数数量,依赖注入一堆服务
func InitGinServer(
	loggerMiddleware *middleware.LoggerMiddleware,
	loginMiddleware *middleware.LoginMiddleware,
	corsMiddleware *middleware.CorsMiddleware,
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
) *gin.Engine {
	//初始化一个gin引擎
	engine := gin.New()
	//全局使用gin中间件
	engine.Use(gin.Recovery())
	api := engine.Group("/api/v1")

	//在所有的中间件之前进行打点路由的注册(这里是给Prometheus读取用的路由),中间件可能导致其失效所以放在最前面
	api.GET("/metrics", gin.WrapH(promhttp.Handler()))

	//使用中间件
	api.Use(
		//gin的默认日志
		gin.Logger(),
		//跨域中间件
		corsMiddleware.MiddlewareFunc(),
		//打点和错误处理中间件
		loggerMiddleware.MiddlewareFunc(),
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
	tube.RegisterRoutes(api, authMiddleware)
	metrics.RegisterRoutes(api, authMiddleware)
	//返回路由
	return engine
}

func timeout() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		_, ok := ctx.Request.Context().Deadline()
		if !ok {
			// 强制给一个超时，省得我前端调试等得不耐烦
			newCtx, cancel := context.WithTimeout(ctx.Request.Context(), time.Second*10)
			defer cancel()
			ctx.Request = ctx.Request.Clone(newCtx)
		}
		ctx.Next()
	}
}
