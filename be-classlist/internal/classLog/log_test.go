package classLog

import (
	"context"
	"github.com/asynccnu/ccnubox-be/be-classlist/test"
	"github.com/go-kratos/kratos/v2/log"
	"testing"
)

func TestWithLogger(t *testing.T) {
	logger := test.NewLogger()
	logger = log.With(logger, "stuId", "testID")
	ctx := context.Background()
	ctx = WithLogger(ctx, logger) // 将 logger 注入到 context 中

	newLogger := GetLoggerFromCtx(ctx)
	h := log.NewHelper(newLogger)
	h.Info("This is a test log message.")
}
