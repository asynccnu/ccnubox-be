package logger

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ZapLogger struct {
	l   *zap.Logger
	ctx context.Context
}

func NewZapLogger(l *zap.Logger) Logger {
	return &ZapLogger{
		l:   l,
		ctx: context.Background(),
	}
}

// WithContext 创建并返回一个持有传入 ctx 的新 zapLogger 指针
// 该 ctx 一般包含 trace 相关信息
func (z *ZapLogger) WithContext(ctx context.Context) Logger {
	return &ZapLogger{
		l:   z.l,
		ctx: ctx,
	}
}

func (z *ZapLogger) Debug(msg string, args ...Field) {
	z.log(zapcore.DebugLevel, msg, args...)
}

func (z *ZapLogger) Info(msg string, args ...Field) {
	z.log(zapcore.InfoLevel, msg, args...)
}

func (z *ZapLogger) Warn(msg string, args ...Field) {
	z.log(zapcore.WarnLevel, msg, args...)
}

// 单独对 Error 级别进行特殊处理，向 Span 报告错误
func (z *ZapLogger) Error(msg string, args ...Field) {
	span := trace.SpanFromContext(z.ctx)

	// 判断是否在 Trace 中
	if span.SpanContext().IsValid() {
		span.RecordError(errors.New(msg))
		span.SetStatus(codes.Error, msg)
	}

	z.log(zapcore.ErrorLevel, msg, args...)
}

// 实现链式字段注入字段，返回 Logger 接口本身
func (z *ZapLogger) With(args ...Field) Logger {
	zapFields := z.toArgs(args)
	newZap := z.l.With(zapFields...)
	return &ZapLogger{
		l:   newZap,
		ctx: z.ctx,
	}
}

// 要直接使用这个功能需要对 Logger 接口直接做断言 zaplogger 使用
func (z *ZapLogger) Sync() error {
	return z.l.Sync()
}

// 这里使用统一的日志处理逻辑负责把 trace_id 和 span_id 注入到 zap 的字段里
func (z *ZapLogger) log(level zapcore.Level, msg string, args ...Field) {
	zapFields := z.toArgs(args)

	// 尝试从 Context 提取 Trace 信息
	span := trace.SpanFromContext(z.ctx)
	if span.SpanContext().IsValid() {
		// 注入 TraceID 和 SpanID
		zapFields = append(zapFields,
			zap.String("trace_id", span.SpanContext().TraceID().String()),
			zap.String("span_id", span.SpanContext().SpanID().String()),
		)
	}

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

func (z *ZapLogger) toArgs(args []Field) []zap.Field {
	res := make([]zap.Field, 0, len(args))
	for _, arg := range args {
		res = append(res, zap.Any(arg.Key, arg.Val))
	}
	return res
}

func ProdEncoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		TimeKey:       "@timestamp",
		LevelKey:      "level",
		MessageKey:    "msg",
		CallerKey:     "caller",
		StacktraceKey: "stacktrace",
		EncodeLevel:   zapcore.CapitalLevelEncoder,
		EncodeTime:    zapcore.ISO8601TimeEncoder,
		EncodeCaller:  zapcore.ShortCallerEncoder,
	}
}

func (z *ZapLogger) Debugf(template string, args ...interface{}) {
	z.AddCallerSkip(1).Debug(fmt.Sprintf(template, args...))
}

func (z *ZapLogger) Infof(template string, args ...interface{}) {
	z.AddCallerSkip(1).Info(fmt.Sprintf(template, args...))
}

func (z *ZapLogger) Warnf(template string, args ...interface{}) {
	z.AddCallerSkip(1).Warn(fmt.Sprintf(template, args...))
}

func (z *ZapLogger) Errorf(template string, args ...interface{}) {
	z.AddCallerSkip(1).Error(fmt.Sprintf(template, args...))
}

func (z *ZapLogger) AddCallerSkip(skip int) Logger {
	newZap := z.l.WithOptions(zap.AddCallerSkip(skip))
	return &ZapLogger{
		l:   newZap,
		ctx: z.ctx,
	}
}
