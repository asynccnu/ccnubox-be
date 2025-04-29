package ioc

import (
	"context"
	"fmt"
	"github.com/asynccnu/ccnubox-be/be-grade/grpc"
	"github.com/asynccnu/ccnubox-be/be-grade/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/be-grade/pkg/grpcx"
	"github.com/asynccnu/ccnubox-be/be-grade/pkg/logger"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport"
	kgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

func InitGRPCxKratosServer(gradeServer *grpc.GradeServiceServer, ecli *clientv3.Client, l logger.Logger) grpcx.Server {
	type Config struct {
		Name    string `yaml:"name"`
		Weight  int    `yaml:"weight"`
		Addr    string `yaml:"addr"`
		EtcdTTL int64  `yaml:"etcdTTL"`
	}
	var cfg Config

	err := viper.UnmarshalKey("grpc.server", &cfg)
	if err != nil {
		panic(err)
	}
	server := kgrpc.NewServer(
		kgrpc.Address(cfg.Addr),
		kgrpc.Middleware(
			recovery.Recovery(),
			LoggingMiddleware(l),
		),
		kgrpc.Timeout(30*time.Second),
	)

	gradeServer.Register(server)
	return &grpcx.KratosServer{
		Server:     server,
		Name:       cfg.Name,
		Weight:     cfg.Weight,
		EtcdTTL:    time.Second * time.Duration(cfg.EtcdTTL),
		EtcdClient: ecli,
		L:          l,
	}
}

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
				customError := errorx.ToCustomError(err)
				if customError != nil {
					// 捕获错误并记录
					l.Error("执行业务出错",
						logger.Error(err),
						logger.String("operationName", operationName),
						logger.String("endPointName", endPointName),
						logger.String("request", fmt.Sprintf("%v", req)),
						logger.String("duration", duration.String()),
						logger.String("timestamp", time.Now().Format(time.RFC3339)),
						logger.String("msg", customError.ERR.Error()),
						logger.String("category", customError.Category),
						logger.String("file", customError.File),
						logger.Int("line", customError.Line),
						logger.String("function", customError.Function),
					)

					//转化为kratos的错误,非常的优雅
					err = customError.ERR
				} else {
					// 记录常规日志
					l.Info("请求成功",
						logger.String("operationName", operationName),
						logger.String("endPointName", endPointName),
						logger.String("request", fmt.Sprintf("%v", req)),
						logger.String("reqHeader", fmt.Sprintf("%v", reqHeader)),
						logger.String("duration", duration.String()),
						logger.String("timestamp", time.Now().Format(time.RFC3339)),
					)
				}

			}

			return reply, err
		}
	}
}
