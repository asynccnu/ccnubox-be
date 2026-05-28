package adapter

import (
	"context"
	"fmt"

	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	klog "github.com/go-kratos/kratos/v2/log"
)

type LoggerFromKratos struct {
	l      klog.Logger
	ctx    context.Context
	fields []logger.Field
}

func NewLoggerFromKratos(l klog.Logger) logger.Logger {
	return &LoggerFromKratos{
		l:   l,
		ctx: context.Background(),
	}
}

func (k *LoggerFromKratos) WithContext(ctx context.Context) logger.Logger {
	return &LoggerFromKratos{
		l:      k.l,
		ctx:    ctx,
		fields: k.fields,
	}
}

func (k *LoggerFromKratos) With(args ...logger.Field) logger.Logger {
	fields := make([]logger.Field, 0, len(k.fields)+len(args))
	fields = append(fields, k.fields...)
	fields = append(fields, args...)

	return &LoggerFromKratos{
		l:      k.l,
		ctx:    k.ctx,
		fields: fields,
	}
}

func (k *LoggerFromKratos) Debug(msg string, args ...logger.Field) {
	klog.NewHelper(k.l).WithContext(k.ctx).Debugw(k.keyvals(msg, args...)...)
}

func (k *LoggerFromKratos) Info(msg string, args ...logger.Field) {
	klog.NewHelper(k.l).WithContext(k.ctx).Infow(k.keyvals(msg, args...)...)
}

func (k *LoggerFromKratos) Warn(msg string, args ...logger.Field) {
	klog.NewHelper(k.l).WithContext(k.ctx).Warnw(k.keyvals(msg, args...)...)
}

func (k *LoggerFromKratos) Error(msg string, args ...logger.Field) {
	klog.NewHelper(k.l).WithContext(k.ctx).Errorw(k.keyvals(msg, args...)...)
}

func (k *LoggerFromKratos) Debugf(template string, args ...interface{}) {
	k.Debug(fmt.Sprintf(template, args...))
}

func (k *LoggerFromKratos) Infof(template string, args ...interface{}) {
	k.Info(fmt.Sprintf(template, args...))
}

func (k *LoggerFromKratos) Warnf(template string, args ...interface{}) {
	k.Warn(fmt.Sprintf(template, args...))
}

func (k *LoggerFromKratos) Errorf(template string, args ...interface{}) {
	k.Error(fmt.Sprintf(template, args...))
}

func (k *LoggerFromKratos) AddCallerSkip(_ int) logger.Logger {
	return k
}

func (k *LoggerFromKratos) keyvals(msg string, args ...logger.Field) []interface{} {
	keyvals := make([]interface{}, 0, 2+(len(k.fields)+len(args))*2)
	keyvals = append(keyvals, "msg", msg)
	for _, field := range k.fields {
		keyvals = append(keyvals, field.Key, field.Val)
	}
	for _, field := range args {
		keyvals = append(keyvals, field.Key, field.Val)
	}
	return keyvals
}
