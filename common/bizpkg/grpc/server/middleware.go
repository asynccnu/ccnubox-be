package server

import (
	"context"
	"fmt"
	"time"

	errorx "github.com/asynccnu/ccnubox-be/common/pkg/errorx/rpcerr"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"go.opentelemetry.io/otel/trace"
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

			var traceId, spanId string
			span := trace.SpanFromContext(ctx)
			if span.SpanContext().IsValid() {
				traceId = span.SpanContext().TraceID().String()
				spanId = span.SpanContext().SpanID().String()
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
				customError := errorx.ToCustomError(err)
				if customError != nil {
					// 捕获错误并记录
					l.Error("执行业务逻辑出错",
						logger.Error(err),
						logger.String("operationName", operationName),
						logger.String("endPointName", endPointName),
						logger.String("request", fmt.Sprintf("%v", req)),
						logger.String("duration", duration.String()),
						logger.String("category", customError.Category),
						logger.String("file", customError.File),
						logger.Int("line", customError.Line),
						logger.String("function", customError.Function),
						logger.String("trace_id", traceId),
						logger.String("span_id", spanId),
					)
					//转化为 kratos 的错误,非常的优雅
					err = customError.ERR
				}
			} else {
				// 记录常规日志
				l.Info("请求成功",
					logger.String("operationName", operationName),
					logger.String("endPointName", endPointName),
					logger.String("request", fmt.Sprintf("%v", req)),
					logger.String("reqHeader", fmt.Sprintf("%v", reqHeader)),
					logger.String("duration", duration.String()),
					logger.String("timestamp", time.Now().Format(time.RFC3339)),
					logger.String("trace_id", traceId),
					logger.String("span_id", spanId),
				)
			}

			return reply, err
		}
	}
}
