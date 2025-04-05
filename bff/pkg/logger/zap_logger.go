package logger

import "go.uber.org/zap"

// ZapLogger 封装了一个 zap.Logger 实例
type ZapLogger struct {
	l *zap.Logger
}

// NewZapLogger 创建一个新的 ZapLogger 实例
// l 是传入的 zap.Logger 实例
func NewZapLogger(l *zap.Logger) Logger {
	return &ZapLogger{
		l: l,
	}
}

// Debug 方法记录一条调试级别的日志消息
// msg 是日志消息
// args 是可变参数，表示附加的字段
func (z *ZapLogger) Debug(msg string, args ...Field) {
	z.l.Debug(msg, z.toArgs(args)...)
}

// Info 方法记录一条信息级别的日志消息
// msg 是日志消息
// args 是可变参数，表示附加的字段
func (z *ZapLogger) Info(msg string, args ...Field) {
	z.l.Info(msg, z.toArgs(args)...)
}

// Warn 方法记录一条警告级别的日志消息
// msg 是日志消息
// args 是可变参数，表示附加的字段
func (z *ZapLogger) Warn(msg string, args ...Field) {
	z.l.Warn(msg, z.toArgs(args)...)
}

// Error 方法记录一条错误级别的日志消息
// msg 是日志消息
// args 是可变参数，表示附加的字段
func (z *ZapLogger) Error(msg string, args ...Field) {
	z.l.Error(msg, z.toArgs(args)...)
}

// toArgs 方法将自定义的 Field 类型转换为 zap.Field 类型
// args 是 Field 类型的切片
// 返回值是 zap.Field 类型的切片
func (z *ZapLogger) toArgs(args []Field) []zap.Field {
	res := make([]zap.Field, 0, len(args))
	for _, arg := range args {
		res = append(res, zap.Any(arg.Key, arg.Val))
	}
	return res
}
