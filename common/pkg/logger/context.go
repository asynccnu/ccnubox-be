package logger

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
)

// 全局单例
var GlobalLogger Logger

func InitGlobalLogger(logger Logger) {
	GlobalLogger = logger
}

type LoggerCtxKey struct{}

func WithLogger(ctx context.Context, Logger Logger) context.Context {
	return context.WithValue(ctx, LoggerCtxKey{}, Logger)
}

func GetLoggerFromCtx(ctx context.Context) Logger {
	ctxLogger, ok := ctx.Value(LoggerCtxKey{}).(Logger)
	if !ok || ctxLogger == nil {
		log.Error("get logger from context failed, using default logger")
		if GlobalLogger != nil {
			return GlobalLogger
		}

		panic("Global logger is not initialized. Call InitGlobalLogger first.")
	}
	return ctxLogger
}
