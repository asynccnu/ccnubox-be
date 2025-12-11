package service

import (
	"context"
	"errors"
	"testing"

	proxyv1 "github.com/asynccnu/ccnubox-be/common/be-api/gen/proto/proxy/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"google.golang.org/grpc"
)

type TestLogger struct {
}

func (t *TestLogger) Debug(msg string, args ...logger.Field) {

}

func (t *TestLogger) Info(msg string, args ...logger.Field) {

}

func (t *TestLogger) Warn(msg string, args ...logger.Field) {

}

func (t *TestLogger) Error(msg string, args ...logger.Field) {

}

func (t *TestLogger) WithContext(ctx context.Context) logger.Logger {
	return &TestLogger{}
}

type MockProxy struct{}

func (m *MockProxy) GetProxyAddr(ctx context.Context, in *proxyv1.GetProxyAddrRequest, opts ...grpc.CallOption) (*proxyv1.GetProxyAddrResponse, error) {
	return nil, errors.New("mock")
}

func Test_ccnuService_getGradCookie(t *testing.T) {
	testLogger := new(TestLogger)
	ccs := NewCCNUService(testLogger, &MockProxy{})
	stuId, password := "", ""
	cookie, err := ccs.GetLibraryCookie(context.Background(), stuId, password)
	if err != nil {
		t.Errorf("GetXKCookie err : %v", err)
	}
	t.Log(cookie)
}
