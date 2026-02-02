package server

import (
	"context"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
)

// LoggingMiddleware 返回一个日志中间件
func LoggingMiddleware(l logger.Logger) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// 获取请求信息
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return handler(ctx, req)
			}

			// 记录请求开始时间
			start := time.Now()

			// 获取调用方信息：服务名称和方法
			operationName := tr.Operation() // 获取调用的服务名称

			endPointName := tr.Endpoint() // 获取调用的具体方法名
			reqHeader := tr.RequestHeader()
			// 执行下一个 handler
			reply, err := handler(ctx, req)

			// 计算耗时
			duration := time.Since(start)

			if err != nil {
				// 捕获错误并记录
				l.WithContext(ctx).Error("执行业务逻辑出错",
					logger.Error(err),
					logger.String("operationName", operationName),
					logger.String("endPointName", endPointName),
					logger.String("request", fmt.Sprintf("%v", req)),
					logger.String("reqHeader", fmt.Sprintf("%v", reqHeader)),
					logger.String("duration", duration.String()),
				)
				//这里会解包获取到存储的grpc的error,这样可以保证服务内的链路不会向外暴露
				err = errorx.Unwrap(err)

			} else {
				// 记录常规日志
				l.WithContext(ctx).Info("请求成功",
					logger.String("operationName", operationName),
					logger.String("endPointName", endPointName),
					logger.String("request", fmt.Sprintf("%v", req)),
					logger.String("reqHeader", fmt.Sprintf("%v", reqHeader)),
					logger.String("duration", duration.String()),
				)
			}

			return reply, err
		}
	}
}
