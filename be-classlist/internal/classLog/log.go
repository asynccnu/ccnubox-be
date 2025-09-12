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
	return ctx.Value(LoggerCtxKey{}).(log.Logger)
}

func GetLogHelperFromCtx(ctx context.Context) *log.Helper {
	logger := GetLoggerFromCtx(ctx)
	return log.NewHelper(logger)
}
