package adapter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"gorm.io/gorm"
	glog "gorm.io/gorm/logger"
)

type GormLogger struct {
	l             logger.Logger
	SlowThreshold time.Duration
	LogLevel      glog.LogLevel
}

func NewGormLogger(l logger.Logger) *GormLogger {
	return &GormLogger{
		l:             l.With(logger.String("scope", "gorm")).AddCallerSkip(2),
		SlowThreshold: 200 * time.Millisecond,
		LogLevel:      glog.Warn,
	}
}

// 并发安全捏
func (g *GormLogger) LogMode(level glog.LogLevel) glog.Interface {
	newLogger := *g
	newLogger.LogLevel = level
	return &newLogger
}

func (g *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if g.LogLevel >= glog.Info {
		g.l.WithContext(ctx).Info(
			// GORM 的 msg 会携带占位符，给他格式化一下
			fmt.Sprintf(msg, data...),
		)
	}
}

func (g *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if g.LogLevel >= glog.Warn {
		g.l.WithContext(ctx).Warn(
			// GORM 的 msg 会携带占位符，给他格式化一下
			fmt.Sprintf(msg, data...),
		)
	}
}

func (g *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if g.LogLevel >= glog.Error {
		g.l.WithContext(ctx).Error(
			// GORM 的 msg 会携带占位符，给他格式化一下
			fmt.Sprintf(msg, data...),
		)
	}
}

// SQL 链路追踪
func (g *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if g.LogLevel <= glog.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	// 构造通用字段
	fields := []logger.Field{
		logger.String("sql", sql),
		logger.Int64("rows", rows),
		logger.String("duration", elapsed.String()),
	}

	l := g.l.WithContext(ctx)

	switch {
	// 发生特殊错误且不是"记录未找到"这种常见错误时报错
	case err != nil && g.LogLevel >= glog.Error && !errors.Is(err, gorm.ErrRecordNotFound):
		l.Error("mysql_error", append(fields, logger.Error(err))...)

	// 慢查询
	case elapsed > g.SlowThreshold && g.SlowThreshold != 0 && g.LogLevel >= glog.Warn:
		slowLogMsg := fmt.Sprintf("slow_sql >= %v", g.SlowThreshold)
		l.Warn(slowLogMsg, fields...)

	// 普通查询（Info 级别打印）
	case g.LogLevel == glog.Info:
		l.Info("mysql_query", fields...)
	}
}
