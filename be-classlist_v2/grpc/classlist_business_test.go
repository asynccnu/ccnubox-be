package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/asynccnu/ccnubox-be/be-classlist_v2/biz/errcode"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/service"
	classlistv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/classlist/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger/zapx"
	"go.uber.org/zap"
)

// 查询课表时，非法学年/学期应在入口被拒绝
func TestGetClass_RejectsInvalidYearSemester(t *testing.T) {
	testLogger := zapx.NewZapLogger(zap.NewNop())
	server := NewCalendarServiceServer(service.NewClasserService(nil, nil, testLogger))

	req := &classlistv1.GetClassRequest{
		StuId:    "2024210001",
		Year:     "1999",
		Semester: "4",
	}

	resp, err := server.GetClass(context.Background(), req)
	t.Logf("response: %+v", resp)
	t.Logf("error: %v", err)

	if !errors.Is(err, errcode.ErrParam) {
		t.Fatalf("expected ErrParam, got %v", err)
	}
}
