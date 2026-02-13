package logger

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type TraceOptions func(*TraceLogger)

// TraceLevel 可以设置链路上报等级
func TraceLevel(level Level) TraceOptions {
	return func(l *TraceLogger) {
		l.level = level
	}
}

type TraceLogger struct {
	logger Logger
	ctx    context.Context
	level  Level
}

func NewTraceLogger(logger Logger, opts ...TraceOptions) Logger {
	l := &TraceLogger{
		logger: logger,
		ctx:    context.Background(),
		level:  ERROR,
	}

	for _, opt := range opts {
		opt(l)
	}

	return l
}

func (f *TraceLogger) WithContext(ctx context.Context) Logger {
	return &TraceLogger{
		logger: f.logger.WithContext(ctx),
		ctx:    ctx,
		level:  f.level,
	}
}

func (f *TraceLogger) With(args ...Field) Logger {
	return &TraceLogger{
		logger: f.logger.With(args...),
		ctx:    f.ctx,
		level:  f.level,
	}
}

func (f *TraceLogger) AddCallerSkip(skip int) Logger {
	return &TraceLogger{
		logger: f.logger.AddCallerSkip(skip),
		ctx:    f.ctx,
		level:  f.level,
	}
}

func (f *TraceLogger) Debug(msg string, args ...Field) {
	f.reportTraceInfo(DEBUG, msg)
	f.logger.Debug(msg, f.addTraceInfo(args)...)
}

func (f *TraceLogger) Debugf(template string, args ...interface{}) {
	f.Debug(fmt.Sprintf(template, args...))
}

func (f *TraceLogger) Info(msg string, args ...Field) {
	f.reportTraceInfo(INFO, msg)
	f.logger.Info(msg, f.addTraceInfo(args)...)
}

func (f *TraceLogger) Infof(template string, args ...interface{}) {
	f.Info(fmt.Sprintf(template, args...))
}

func (f *TraceLogger) Warn(msg string, args ...Field) {
	f.reportTraceInfo(WARN, msg)
	f.logger.Warn(msg, f.addTraceInfo(args)...)
}

func (f *TraceLogger) Warnf(template string, args ...interface{}) {
	f.Warn(fmt.Sprintf(template, args...))
}

func (f *TraceLogger) Error(msg string, args ...Field) {
	f.reportTraceInfo(ERROR, msg)
	f.logger.Error(msg, f.addTraceInfo(args)...)
}

func (f *TraceLogger) Errorf(template string, args ...interface{}) {
	f.Error(fmt.Sprintf(template, args...))
}

func (f *TraceLogger) reportTraceInfo(level Level, msg string) {
	span := trace.SpanFromContext(f.ctx)

	// 如果在span中,同时大于等于当前的上报等级则上报
	if span.SpanContext().IsValid() && level >= f.level {
		span.RecordError(errors.New(msg))
		if level >= ERROR {
			span.SetStatus(codes.Error, msg)
		} else {
			span.SetStatus(codes.Ok, msg)
		}
	}
}

// 自动添加链路信息到日志中去
func (f *TraceLogger) addTraceInfo(fields []Field) []Field {
	span := trace.SpanFromContext(f.ctx)
	if span.SpanContext().IsValid() {
		// 注入 TraceID 和 SpanID
		fields = append(fields,
			String("trace_id", span.SpanContext().TraceID().String()),
			String("span_id", span.SpanContext().SpanID().String()),
		)
	}

	return fields
}
