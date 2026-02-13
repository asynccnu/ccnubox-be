package log

import (
	"github.com/asynccnu/ccnubox-be/common/bizpkg/conf"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger/zapx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func InitLogger(cfg *conf.LogConf, skip int) logger.Logger {
	// 直接使用 zapx 本身的配置结构体来处理
	// 配置Lumberjack以支持日志文件的滚动

	lumberjackLogger := &lumberjack.Logger{
		// 注意有没有权限
		Filename:   cfg.Path,       // 指定日志文件路径
		MaxSize:    cfg.MaxSize,    // 每个日志文件的最大大小，单位：MB
		MaxBackups: cfg.MaxBackups, // 保留旧日志文件的最大个数
		MaxAge:     cfg.MaxAge,     // 保留旧日志文件的最大天数
		Compress:   cfg.Compress,   // 是否压缩旧的日志文件
	}

	// 创建zap日志核心
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zapx.ProdEncoderConfig()),
		zapcore.AddSync(lumberjackLogger),
		zapcore.DebugLevel, // 设置日志级别
	)

	l := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel), zap.AddCallerSkip(skip))
	res := zapx.NewZapLogger(l)

	// 这里默认会用带链路的日志
	res = logger.NewTraceLogger(res, logger.TraceLevel(logger.ERROR))
	logger.InitGlobalLogger(res)
	return res
}
