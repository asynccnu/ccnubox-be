package middleware

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/asynccnu/ccnubox-be/bff/errs"
	b_errorx "github.com/asynccnu/ccnubox-be/bff/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/web"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/gin-gonic/gin"
)

type LoggerMiddleware struct {
	log logger.Logger
}

func NewLoggerMiddleware(
	log logger.Logger,
) *LoggerMiddleware {
	return &LoggerMiddleware{
		log: log,
	}
}

func (lm *LoggerMiddleware) MiddlewareFunc() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Next() // 执行后续逻辑

		// 处理返回值或错误
		res, httpCode := lm.handleResponse(ctx)
		if !ctx.IsAborted() { // 避免重复返回响应
			ctx.JSON(httpCode, res)
		}
	}
}

// 处理响应逻辑
func (lm *LoggerMiddleware) handleResponse(ctx *gin.Context) (web.Response, int) {
	var res web.Response
	httpCode := ctx.Writer.Status()

	// 有错误则进行错误处理
	if len(ctx.Errors) > 0 {
		// http层error,携带httpCode,bizCode,msg
		err := ctx.Errors.Last().Err
		unwarpERR := errors.Unwrap(err)
		if unwarpERR == nil {
			lm.log.WithContext(ctx).Error("意外错误类型",
				logger.Error(err),
				logger.String("ip", ctx.ClientIP()),
				logger.String("path", ctx.Request.URL.Path),
				logger.String("method", ctx.Request.Method),
				logger.String("headers", fmt.Sprintf("%v", ctx.Request.Header)),
			)
			return web.Response{Code: errs.ERROR_TYPE_ERROR_CODE, Msg: err.Error(), Data: nil}, http.StatusInternalServerError
		}

		bizErr, ok := unwarpERR.(*b_errorx.CustomError)
		if !ok {
			lm.log.WithContext(ctx).Error("意外错误类型",
				logger.Error(err),
				logger.String("ip", ctx.ClientIP()),
				logger.String("path", ctx.Request.URL.Path),
				logger.String("method", ctx.Request.Method),
				logger.String("headers", fmt.Sprintf("%v", ctx.Request.Header)),
			)
			return web.Response{Code: errs.ERROR_TYPE_ERROR_CODE, Msg: err.Error(), Data: nil}, http.StatusInternalServerError

		}

		lm.log.WithContext(ctx).Error("处理请求出错",
			logger.Error(bizErr), // bizErr
			logger.String("ip", ctx.ClientIP()),
			logger.String("path", ctx.Request.URL.Path),
			logger.String("method", ctx.Request.Method),
			logger.String("headers", fmt.Sprintf("%v", ctx.Request.Header)),
			logger.Int("httpCode", bizErr.HttpCode),
			logger.Int("code", bizErr.Code),
			logger.String("msg", bizErr.Message),
		)
		return web.Response{Code: bizErr.Code, Msg: bizErr.Message, Data: nil}, bizErr.HttpCode
	}

	// 无错误则记录常规日志
	lm.log.WithContext(ctx).Info("请求正常",
		logger.String("ip", ctx.ClientIP()),
		logger.String("path", ctx.Request.URL.Path),
		logger.String("method", ctx.Request.Method),
		logger.String("headers", fmt.Sprintf("%v", ctx.Request.Header)),
	)
	res = ginx.GetResp[web.Response](ctx)

	// 用来保证gin中间件实现404的时候也能有消息提示
	if httpCode == http.StatusNotFound {
		res.Msg = "不存在的路由或请求方法!"
	}

	return res, httpCode
}
