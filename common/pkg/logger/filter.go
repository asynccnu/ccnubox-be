package logger

import (
	"context"
	"fmt"

	klog "github.com/go-kratos/kratos/v2/log"
)

const fuzzyStr = "***"

type FilterOptions func(*FilterLogger)

func FilterKey(keys ...string) FilterOptions {
	return func(fl *FilterLogger) {
		for _, key := range keys {
			fl.filterKeys[key] = struct{}{}
		}
	}
}

func FilterValue(value ...string) FilterOptions {
	return func(fl *FilterLogger) {
		for _, val := range value {
			fl.filterVals[val] = struct{}{}
		}
	}
}

func FilterFunc(f func(level Level, key, val string) (string, bool)) FilterOptions {
	return func(fl *FilterLogger) {
		fl.filterFuncSlice = append(fl.filterFuncSlice, f)
	}
}

type FilterLogger struct {
	logger     Logger
	filterKeys map[string]struct{}
	filterVals map[string]struct{}

	filterFuncSlice []func(level Level, key, val string) (string, bool)
}

func NewFilterLogger(logger Logger, opts ...FilterOptions) Logger {
	fl := &FilterLogger{
		logger:     logger,
		filterKeys: make(map[string]struct{}),
		filterVals: make(map[string]struct{}),
	}
	for _, opt := range opts {
		opt(fl)
	}
	return fl
}

func (f *FilterLogger) filter(level Level, fields []Field) []Field {
	if len(fields) == 0 {
		return fields
	}

	out := make([]Field, 0, len(fields))
	for _, field := range fields {
		if fuzzy, ok := f.shouldFilter(level, field); ok {
			out = append(out, Field{
				Key: field.Key,
				Val: fuzzy,
			})
			continue
		}
		out = append(out, field)
	}
	return out
}

// 检查是否需要过滤该字段,如果需要过滤则返回模糊字符串，否则返回空字符串（可忽略）
func (f *FilterLogger) shouldFilter(level Level, field Field) (string, bool) {
	if _, ok := f.filterKeys[field.Key]; ok {
		return fuzzyStr, true
	}

	if v, ok := stringify(field.Val); ok {
		if _, hit := f.filterVals[v]; hit {
			return fuzzyStr, true
		}

		if len(f.filterFuncSlice) > 0 {
			for _, fn := range f.filterFuncSlice {
				if newFuzzyStr, shouldFuzzy := fn(level, field.Key, v); shouldFuzzy {
					return newFuzzyStr, true
				}
			}
		}

	}
	return "", false
}

func stringify(val any) (string, bool) {
	switch v := val.(type) {
	case string:
		return v, true
	case fmt.Stringer:
		return v.String(), true
	default:
		return "", false
	}
}

func (f *FilterLogger) WithContext(ctx context.Context) Logger {
	return &FilterLogger{
		logger:          f.logger.WithContext(ctx),
		filterKeys:      f.filterKeys,
		filterVals:      f.filterVals,
		filterFuncSlice: f.filterFuncSlice,
	}
}

func (f *FilterLogger) Debug(msg string, args ...Field) {
	f.logger.Debug(msg, f.filter(DEBUG, args)...)
}

func (f *FilterLogger) Info(msg string, args ...Field) {
	f.logger.Info(msg, f.filter(INFO, args)...)
}

func (f *FilterLogger) Warn(msg string, args ...Field) {
	f.logger.Warn(msg, f.filter(WARN, args)...)
}

func (f *FilterLogger) Error(msg string, args ...Field) {
	f.logger.Error(msg, f.filter(ERROR, args)...)
}

func (f *FilterLogger) With(args ...Field) Logger {
	filteredArgs := f.filter(INFO, args)
	newBaseLogger := f.logger.With(filteredArgs...)

	return &FilterLogger{
		logger:          newBaseLogger,
		filterKeys:      f.filterKeys,
		filterVals:      f.filterVals,
		filterFuncSlice: f.filterFuncSlice,
	}
}

// Log 方法只是为了实现 Kratos 内部的
func (f *FilterLogger) Log(level klog.Level, keyvals ...any) error {
	// 如果没有参数，直接调用底层
	if len(keyvals) == 0 {
		return f.logger.Log(level, keyvals...)
	}

	// 重新构建过滤后的 keyvals
	// 预分配切片容量
	out := make([]any, 0, len(keyvals))

	// 存入kv
	for i := 0; i < len(keyvals); i += 2 {
		key := fmt.Sprint(keyvals[i])

		// 为避免传入奇数个参数时存在某个 key 的 value 为空
		var val any = "MISSING_VALUE"
		if i+1 < len(keyvals) {
			val = keyvals[i+1]
		}

		_, ok := f.filterKeys[key]
		if ok {
			out = append(out, key, fuzzyStr)
			continue
		}

		strVal, ok := stringify(val)
		if ok {
			_, hit := f.filterVals[strVal]
			if hit {
				out = append(out, key, fuzzyStr)
				continue
			}

			filtered := false

			for _, fn := range f.filterFuncSlice {
				newVal, hit := fn(toSelfLevel(level), key, strVal)
				if hit {
					out = append(out, key, newVal)
					filtered = true
					break
				}
			}
			
			if filtered {
				continue
			}
		}
		out = append(out, key, val)
	}

	return f.logger.Log(level, out...)
}
