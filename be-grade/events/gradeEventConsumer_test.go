package events

import (
	"context"
	"errors"
	"testing"

	"github.com/asynccnu/ccnubox-be/be-grade/domain"
	"github.com/asynccnu/ccnubox-be/be-grade/service"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger/zapx"
	"go.uber.org/zap"
)

type testGradeService struct {
	service.GradeService
	calls int
	ctx   context.Context
	err   error
}

func (s *testGradeService) UpdateDetailScore(ctx context.Context, _ domain.NeedDetailGrade) error {
	s.calls++
	s.ctx = ctx
	if s.calls == 1 {
		return s.err
	}
	return nil
}

func TestGradeConsumePropagatesPartialFailure(t *testing.T) {
	failure := errors.New("update failed")
	svc := &testGradeService{err: failure}
	h := &GradeDetailEventConsumerHandler{gradeService: svc, l: zapx.NewZapLogger(zap.NewNop())}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := []domain.NeedDetailGrade{{StudentID: "first"}, {StudentID: "second"}}
	if err := h.consume(ctx, events); !errors.Is(err, failure) {
		t.Fatalf("consume() = %v, want update error", err)
	}
	if svc.calls != 2 || svc.ctx != ctx {
		t.Fatalf("calls=%d context=%v", svc.calls, svc.ctx)
	}
	if err := h.consume(ctx, events); err != nil {
		t.Fatalf("retry after recovery: %v", err)
	}
	cancel()
	if err := h.consume(ctx, events); !errors.Is(err, context.Canceled) || svc.calls != 4 {
		t.Fatalf("cancelled consume: err=%v calls=%d", err, svc.calls)
	}
}
