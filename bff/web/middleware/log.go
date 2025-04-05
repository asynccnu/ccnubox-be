package middleware

import (
	"fmt"
	"github.com/asynccnu/ccnubox-be/bff/errs"
	"github.com/asynccnu/ccnubox-be/bff/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/pkg/logger"
	"github.com/asynccnu/ccnubox-be/bff/pkg/prometheusx"
	"github.com/asynccnu/ccnubox-be/bff/web"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

type LoggerMiddleware struct {
	log        logger.Logger
	prometheus *prometheusx.PrometheusCounter
}

func NewLoggerMiddleware(
	log logger.Logger,
	prometheus *prometheusx.PrometheusCounter,
) *LoggerMiddleware {
	return &LoggerMiddleware{
		log:        log,
		prometheus: prometheus,
	}
}

func (lm *LoggerMiddleware) MiddlewareFunc() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()
		path := ctx.FullPath()

		// 记录活跃连接数
		lm.prometheus.ActiveConnections.WithLabelValues(path).Inc()
		defer func() {
			//打点路由特殊化处理这里还没有想到更好的方案,先这样吧
			if path == "/api/v1/metrics/:eventName" {
				path = "/api/v1/metrics/" + ctx.Param("eventName")
			}
			// 记录响应信息
			lm.prometheus.ActiveConnections.WithLabelValues(path).Dec()
			status := ctx.Writer.Status()
			lm.prometheus.RouterCounter.WithLabelValues(ctx.Request.Method, path, http.StatusText(status)).Inc()
			lm.prometheus.DurationTime.WithLabelValues(path, http.StatusText(status)).Observe(time.Since(start).Seconds())
		}()

		ctx.Next() // 执行后续逻辑

		// 处理返回值或错误
		res, httpCode := lm.handleResponse(ctx)
		if !ctx.IsAborted() { // 避免重复返回响应
			ctx.JSON(httpCode, res)
		}
	}
}

// 提取的日志逻辑：记录自定义错误日志
func (lm *LoggerMiddleware) logCustomError(customError *errorx.CustomError, ctx *gin.Context) {
	lm.log.Error("处理请求出错",
		logger.Error(customError),
		logger.String("timestamp", time.Now().Format(time.RFC3339)),
		logger.String("ip", ctx.ClientIP()),
		logger.String("path", ctx.Request.URL.Path),
		logger.String("method", ctx.Request.Method),
		logger.String("headers", fmt.Sprintf("%v", ctx.Request.Header)),
		logger.Int("httpCode", customError.HttpCode),
		logger.Int("code", customError.Code),
		logger.String("msg", customError.Msg),
		logger.String("category", customError.Category),
		logger.String("file", customError.File),
		logger.Int("line", customError.Line),
		logger.String("function", customError.Function),
	)
}

// 提取的日志逻辑：记录未知错误日志
func (lm *LoggerMiddleware) logUnexpectedError(err error, ctx *gin.Context) {
	lm.log.Error("意外错误类型",
		logger.Error(err),
		logger.String("timestamp", time.Now().Format(time.RFC3339)),
		logger.String("ip", ctx.ClientIP()),
		logger.String("path", ctx.Request.URL.Path),
		logger.String("method", ctx.Request.Method),
		logger.String("headers", fmt.Sprintf("%v", ctx.Request.Header)),
	)
}

func (lm *LoggerMiddleware) commonInfo(ctx *gin.Context) {
	lm.log.Info("意外错误类型",
		logger.String("timestamp", time.Now().Format(time.RFC3339)),
		logger.String("ip", ctx.ClientIP()),
		logger.String("path", ctx.Request.URL.Path),
		logger.String("method", ctx.Request.Method),
		logger.String("headers", fmt.Sprintf("%v", ctx.Request.Header)),
	)
}

// 处理响应逻辑
func (lm *LoggerMiddleware) handleResponse(ctx *gin.Context) (web.Response, int) {
	var res web.Response
	httpCode := ctx.Writer.Status()

	//有错误则进行错误处理
	if len(ctx.Errors) > 0 {
		err := ctx.Errors.Last().Err
		customError := errorx.ToCustomError(err)
		if customError == nil {
			lm.logUnexpectedError(err, ctx)
			return web.Response{Code: errs.ERROR_TYPE_ERROR_CODE, Msg: err.Error(), Data: nil}, http.StatusInternalServerError
		}
		lm.logCustomError(customError, ctx)
		return web.Response{Code: customError.Code, Msg: customError.Msg, Data: nil}, customError.HttpCode
	} else {

		//无错误则记录常规日志
		lm.commonInfo(ctx)
		res = ginx.GetResp[web.Response](ctx)
	}

	//用来保证gin中间件实现404的时候也能有消息提示
	if httpCode == http.StatusNotFound {
		res.Msg = "不存在的路由或请求方法!"
	}

	return res, httpCode
}
