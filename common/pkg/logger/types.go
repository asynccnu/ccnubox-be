package logger

import (
	"context"
)

type Logger interface {
	WithContext(ctx context.Context) Logger
	With(args ...Field) Logger

	Debug(msg string, args ...Field)
	Info(msg string, args ...Field)
	Warn(msg string, args ...Field)
	Error(msg string, args ...Field)

	Debugf(template string, args ...interface{})
	Infof(template string, args ...interface{})
	Warnf(template string, args ...interface{})
	Errorf(template string, args ...interface{})

	AddCallerSkip(skip int) Logger
}

type Field struct {
	Key string
	Val any
}

func Any(key string, val any) Field {
	return Field{
		Key: key,
		Val: val,
	}
}

func Error(err error) Field {
	return Field{
		Key: "error",
		Val: err.Error(),
	}
}

func Int64(key string, val int64) Field {
	return Field{
		Key: key,
		Val: val,
	}
}

func Int(key string, val int) Field {
	return Field{
		Key: key,
		Val: val,
	}
}

func String(key string, val string) Field {
	return Field{
		Key: key,
		Val: val,
	}
}

func Int32(key string, val int32) Field {
	return Field{
		Key: key,
		Val: val,
	}
}
