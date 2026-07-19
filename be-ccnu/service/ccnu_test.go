package service

import (
	"context"
	"testing"

	ccnuv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/ccnu/v1"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/proxy"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger/zapx"
	"go.uber.org/zap"
)

func Test_ccnuService_getGradCookie(t *testing.T) {
	testLogger := zapx.NewZapLogger(zap.NewNop())
	ccs := NewCCNUService(testLogger, proxy.NewDirectHttpProxy(nil), "test-secret")
	stuId, password := "", ""
	_, err := ccs.GetLibraryToken(context.Background(), stuId, password, ccnuv1.LIBRARY_TYPE_LIBRARY_SEAT)
	if err == nil {
		t.Fatal("expected invalid student id error")
	}
}
