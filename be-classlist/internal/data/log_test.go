package data

import (
	"context"
	"testing"

	"github.com/asynccnu/ccnubox-be/common/bizpkg/conf"
	bizlog "github.com/asynccnu/ccnubox-be/common/bizpkg/log"

	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

var testLogConf = &conf.LogConf{
	Path:       "../logs/app.log",
	MaxSize:    100,
	MaxBackups: 7,
	MaxAge:     30,
	Compress:   true,
}

func TestWithLogger(t *testing.T) {
	testlogger := bizlog.InitLogger(testLogConf)
	testlogger = testlogger.With(
		logger.String("stu_id", "testId"),
	)
	ctx := context.Background()

	ctx = logger.WithLogger(ctx, testlogger) // 将 logger 注入到 context 中
	newLogger := logger.GetLoggerFromCtx(ctx).(*logger.ZapLogger)
	defer newLogger.Sync()

	newLogger.Info("test",
		logger.String("hellokey", "worldvalue"),
	)
}
