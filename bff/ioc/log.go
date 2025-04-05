package ioc

import (
	"github.com/asynccnu/ccnubox-be/bff/pkg/logger"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func InitLogger() logger.Logger {
	// 直接使用 zap 本身的配置结构体来处理
	// 配置Lumberjack以支持日志文件的滚动

	var cfg struct {
		Path       string `yaml:"path"`
		MaxSize    int    `yaml:"maxSize"`    // 每个日志文件的最大大小，单位：MB
		MaxBackups int    `yaml:"maxBackups"` // 保留旧日志文件的最大个数
		MaxAge     int    `yaml:"maxAge"`     // 保留旧日志文件的最大天数
		Compress   int    `yaml:"compress"`   // 是否压缩旧的日志文件
	}

	if err := viper.UnmarshalKey("log", &cfg); err != nil {
		panic(err)
	}

	lumberjackLogger := &lumberjack.Logger{
		// 注意有没有权限
		Filename:   cfg.Path,       // 指定日志文件路径
		MaxSize:    cfg.MaxSize,    // 每个日志文件的最大大小，单位：MB
		MaxBackups: cfg.MaxBackups, // 保留旧日志文件的最大个数
		MaxAge:     cfg.MaxAge,     // 保留旧日志文件的最大天数
		Compress:   true,           // 是否压缩旧的日志文件
	}

	// 创建zap日志核心
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(lumberjackLogger),
		zapcore.DebugLevel, // 设置日志级别
	)

	l := zap.New(core, zap.AddCaller())
	res := logger.NewZapLogger(l)

	return res
}
