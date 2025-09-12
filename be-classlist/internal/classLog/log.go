package classLog

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
)

type LoggerCtxKey struct{}

func WithLogger(ctx context.Context, logger log.Logger) context.Context {
	return context.WithValue(ctx, LoggerCtxKey{}, logger)
}

func GetLoggerFromCtx(ctx context.Context) log.Logger {
	logger, ok := ctx.Value(LoggerCtxKey{}).(log.Logger)
	if !ok || logger == nil {
		log.Error("get logger from context failed, using default logger")
		return log.DefaultLogger
	}
	return logger
}

func GetLogHelperFromCtx(ctx context.Context) *log.Helper {
	logger := GetLoggerFromCtx(ctx)
	return log.NewHelper(logger)
}
