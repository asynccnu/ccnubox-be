package health

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// errorHealthClient 下游返回错误且响应为 nil 的桩客户端。
type errorHealthClient struct {
	healthpb.HealthClient
	err error
}

func (c *errorHealthClient) Check(_ context.Context, _ *healthpb.HealthCheckRequest, _ ...grpc.CallOption) (*healthpb.HealthCheckResponse, error) {
	return nil, c.err
}

func TestReadyCheckDownstreamError(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("ReadyCheck panicked on downstream error: %v", r)
		}
	}()

	downstreamErr := errors.New("connection refused")
	handler := NewHealthHandler(map[string]healthpb.HealthClient{
		"grade": &errorHealthClient{err: downstreamErr},
	})
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

	resp, err := handler.ReadyCheck(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != 200 {
		t.Fatalf("resp.Code = %d, want 200", resp.Code)
	}
	data, ok := resp.Data.(map[string]string)
	if !ok {
		t.Fatalf("resp.Data type = %T, want map[string]string", resp.Data)
	}
	if len(data) != 1 {
		t.Fatalf("resp.Data = %+v, want only the failing service", data)
	}
	if got := data["grade"]; got != "connection refused" {
		t.Fatalf("data[grade] = %q, want %q", got, "connection refused")
	}
}
