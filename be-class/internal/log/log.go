package log

import (
	kzap "github.com/go-kratos/kratos/contrib/log/zap/v2"
	"github.com/go-kratos/kratos/v2/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

var LogPrinter = NewLogger()

func NewLogger() *log.Helper {
	return log.NewHelper(kzap.NewLogger(newZapLogger()))
}

func newZapLogger() *zap.Logger {
	writeSyncer, err := getLogWriter() // 日志文件配置 文件位置和切割
	if err != nil {
		return nil
	}
	encoder := getEncoder() // 获取日志输出编码
	level := zapcore.InfoLevel
	core := zapcore.NewCore(encoder, writeSyncer, level)
	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(2), zap.AddStacktrace(zapcore.ErrorLevel))
	return logger
}

// getEncoder 编码器(如何写入日志)
func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder   // log 时间格式 例如: 2021-09-11t20:05:54.852+0800
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder // 输出level序列化为全大写字符串，如 INFO DEBUG ERROR
	//encoderConfig.EncodeCaller = zapcore.FullCallerEncoder
	//encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	return zapcore.NewConsoleEncoder(encoderConfig) // 以logfmt格式写入
}

// getLogWriter 获取日志输出方式  控制台
func getLogWriter() (zapcore.WriteSyncer, error) {
	return zapcore.AddSync(os.Stdout), nil
}
