package logger

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
)

// TODO(classlist-v2): keep this compatibility layer until v1 services finish
// migrating to constructor-injected logger.Logger, then delete this file.

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

// 给在业务逻辑中获取 logger 提供一个标准方法
// 当业务上层注入该日志链路其所特殊的字段时，logger 将会被修饰，需要被注入到上下文中传递
// 所以在后继链路需要从上下文中提取被修饰过的 logger
// 若上下文不存在被修饰过的 logger，则取默认 logger 注入当前 ctx（存有链路信息）得到新带有链路信息的 logger
func From(ctx context.Context) Logger {
	if l := GetLoggerFromCtx(ctx); l != nil {
		return l
	}
	return GlobalLogger.WithContext(ctx)
}
