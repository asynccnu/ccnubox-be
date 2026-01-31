package classLog

import (
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/conf"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/pkg/tool"
	kzap "github.com/go-kratos/kratos/contrib/log/zap/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"path/filepath"
)

func Logger(c *conf.ZapLogConfigs) log.Logger {
	return kzap.NewLogger(NewZapLogger(c))
}

func NewZapLogger(c *conf.ZapLogConfigs) *zap.Logger {
	logLevel := map[string]zapcore.Level{
		"debug": zapcore.DebugLevel,
		"info":  zapcore.InfoLevel,
		"warn":  zapcore.WarnLevel,
		"error": zapcore.ErrorLevel,
	}
	writeSyncer, err := getLogWriter(c) // 日志文件配置 文件位置和切割
	if err != nil {
		return nil
	}
	encoder := getEncoder(c)          // 获取日志输出编码
	level, ok := logLevel[c.LogLevel] // 日志打印级别
	if !ok {
		level = logLevel["info"]
	}
	core := zapcore.NewCore(encoder, writeSyncer, level)
	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(3), zap.AddStacktrace(zapcore.ErrorLevel)) // classLog.Addcaller() 输出日志打印文件和行数如： classLog/logger_test.go:33
	return logger
}

// getEncoder 编码器(如何写入日志)
func getEncoder(conf *conf.ZapLogConfigs) zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder   // log 时间格式 例如: 2021-09-11t20:05:54.852+0800
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder // 输出level序列化为全大写字符串，如 INFO DEBUG ERROR
	//encoderConfig.EncodeCaller = zapcore.FullCallerEncoder
	//encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	if conf.LogFormat == "json" {
		return zapcore.NewJSONEncoder(encoderConfig) // 以json格式写入
	}
	return zapcore.NewConsoleEncoder(encoderConfig) // 以logfmt格式写入
}

// getLogWriter 获取日志输出方式  日志文件 控制台
func getLogWriter(conf *conf.ZapLogConfigs) (zapcore.WriteSyncer, error) {

	// 判断日志路径是否存在，如果不存在就创建
	if exist := tool.IsExist(conf.LogPath); !exist {
		if err := os.MkdirAll(conf.LogPath, os.ModePerm); err != nil {
			return nil, err
		}
	}

	// 日志文件 与 日志切割 配置
	lumberJackLogger := NewLumberjackLogger(conf.LogPath, conf.LogFileName, int(conf.LogFileMaxSize), int(conf.LogFileMaxBackups), int(conf.LogMaxAge), conf.LogCompress)
	if conf.LogStdout {
		// 日志同时输出到控制台和日志文件中
		return zapcore.NewMultiWriteSyncer(zapcore.AddSync(lumberJackLogger), zapcore.AddSync(os.Stdout)), nil
	} else {
		// 日志只输出到日志文件
		return zapcore.AddSync(lumberJackLogger), nil
	}
}

func NewLumberjackLogger(logPath, logFileName string, fileMaxSize, logFileMaxBackups, logMaxAge int, logCompress bool) *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:   filepath.Join(logPath, logFileName), // 日志文件路径
		MaxSize:    fileMaxSize,                         // 单个日志文件最大多少 mb
		MaxBackups: logFileMaxBackups,                   // 日志备份数量
		MaxAge:     logMaxAge,                           // 日志最长保留时间
		Compress:   logCompress,                         // 是否压缩日志
	}
}
