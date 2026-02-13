package zapx

import (
	"context"
	"fmt"

	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ZapLogger struct {
	l   *zap.Logger
	ctx context.Context
}

func NewZapLogger(l *zap.Logger) logger.Logger {
	return &ZapLogger{
		l:   l,
		ctx: context.Background(),
	}
}

// WithContext 创建并返回一个持有传入 ctx 的新 zapLogger 指针
func (z *ZapLogger) WithContext(ctx context.Context) logger.Logger {
	return &ZapLogger{
		l:   z.l,
		ctx: ctx,
	}
}

func (z *ZapLogger) Debug(msg string, args ...logger.Field) {
	z.log(zapcore.DebugLevel, msg, args...)
}

func (z *ZapLogger) Debugf(template string, args ...interface{}) {
	z.Debug(fmt.Sprintf(template, args...))
}

func (z *ZapLogger) Info(msg string, args ...logger.Field) {
	z.log(zapcore.InfoLevel, msg, args...)
}

func (z *ZapLogger) Infof(template string, args ...interface{}) {
	z.Info(fmt.Sprintf(template, args...))
}

func (z *ZapLogger) Warn(msg string, args ...logger.Field) {
	z.log(zapcore.WarnLevel, msg, args...)
}

func (z *ZapLogger) Warnf(template string, args ...interface{}) {
	z.Warn(fmt.Sprintf(template, args...))
}

// 单独对 Error 级别进行特殊处理，向 Span 报告错误
func (z *ZapLogger) Error(msg string, args ...logger.Field) {
	z.log(zapcore.ErrorLevel, msg, args...)
}

func (z *ZapLogger) Errorf(template string, args ...interface{}) {
	z.Error(fmt.Sprintf(template, args...))
}

func (z *ZapLogger) With(args ...logger.Field) logger.Logger {
	zapFields := z.toArgs(args)
	return &ZapLogger{
		l:   z.l.With(zapFields...),
		ctx: z.ctx,
	}
}

func (z *ZapLogger) AddCallerSkip(skip int) logger.Logger {
	return &ZapLogger{
		l:   z.l.WithOptions(zap.AddCallerSkip(skip)),
		ctx: z.ctx,
	}
}

// 这里使用统一的日志处理逻辑负责把 trace_id 和 span_id 注入到 zapx 的字段里
func (z *ZapLogger) log(level zapcore.Level, msg string, args ...logger.Field) {
	zapFields := z.toArgs(args)

	switch level {
	case zapcore.DebugLevel:
		z.l.Debug(msg, zapFields...)
	case zapcore.InfoLevel:
		z.l.Info(msg, zapFields...)
	case zapcore.WarnLevel:
		z.l.Warn(msg, zapFields...)
	case zapcore.ErrorLevel:
		z.l.Error(msg, zapFields...)
	default:
		z.l.Info(msg, zapFields...)
	}
}

func (z *ZapLogger) toArgs(args []logger.Field) []zap.Field {
	res := make([]zap.Field, 0, len(args))
	for _, arg := range args {
		res = append(res, zap.Any(arg.Key, arg.Val))
	}
	return res
}

func ProdEncoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		TimeKey:      "@timestamp",
		LevelKey:     "level",
		MessageKey:   "msg",
		CallerKey:    "caller",
		EncodeLevel:  zapcore.CapitalLevelEncoder,
		EncodeTime:   zapcore.ISO8601TimeEncoder,
		EncodeCaller: zapcore.ShortCallerEncoder,
	}
}
