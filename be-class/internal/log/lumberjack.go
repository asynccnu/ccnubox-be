package log

import (
	"github.com/natefinch/lumberjack"
	"path/filepath"
)

func NewLumberjackLogger(logPath, logFileName string, fileMaxSize, logFileMaxBackups, logMaxAge int, logCompress bool) *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:   filepath.Join(logPath, logFileName), // 日志文件路径
		MaxSize:    fileMaxSize,                         // 单个日志文件最大多少 mb
		MaxBackups: logFileMaxBackups,                   // 日志备份数量
		MaxAge:     logMaxAge,                           // 日志最长保留时间
		Compress:   logCompress,                         // 是否压缩日志
	}
}
