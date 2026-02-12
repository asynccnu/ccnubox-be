package adapter

import (
	"fmt"

	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	klog "github.com/go-kratos/kratos/v2/log"
)

// 兼容 kratos 框架
// 对 kratos 框架的自身日志做的 logger adapter
type KratosLogger struct {
	l logger.Logger
}

func NewKratosLogger(l logger.Logger) klog.Logger {
	return &KratosLogger{
		l: l.With(logger.String("scope", "kratos")).AddCallerSkip(3),
	}
}

func (k *KratosLogger) Log(level klog.Level, keyvals ...any) error {
	fields := make([]logger.Field, 0, len(keyvals)/2)
	var msg string

	for i := 0; i < len(keyvals); i += 2 {
		key := fmt.Sprint(keyvals[i])
		var val any
		if i+1 < len(keyvals) {
			val = keyvals[i+1]
		}

		if key == "msg" || key == "message" {
			msg = fmt.Sprint(val)
			continue
		}
		fields = append(fields, logger.Any(key, val))
	}

	switch level {
	case klog.LevelDebug:
		k.l.Debug(msg, fields...)
	case klog.LevelInfo:
		k.l.Info(msg, fields...)
	case klog.LevelWarn:
		k.l.Warn(msg, fields...)
	case klog.LevelError:
		k.l.Error(msg, fields...)
	case klog.LevelFatal:
		k.l.Error(msg, fields...)
	default:
		k.l.Info(msg, fields...)
	}

	return nil
}
