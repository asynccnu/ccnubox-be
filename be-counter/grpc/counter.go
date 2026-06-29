package grpc

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-counter/service"
	counterv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/counter/v1"
	"google.golang.org/grpc"
)

type CounterServiceServer struct {
	counterv1.UnimplementedCounterServiceServer
	svc service.CounterService
}

func NewCounterServiceServer(svc service.CounterService) *CounterServiceServer {
	return &CounterServiceServer{svc: svc}
}

func (d *CounterServiceServer) AddCounter(ctx context.Context, req *counterv1.AddCounterReq) (*counterv1.AddCounterResp, error) {
	err := d.svc.AddCounter(ctx, req.GetStudentId())
	if err != nil {
		return nil, err
	}
	return &counterv1.AddCounterResp{}, nil
}

func (d *CounterServiceServer) GetCounterLevels(ctx context.Context, req *counterv1.GetCounterLevelsReq) (*counterv1.GetCounterLevelsResp, error) {
	ids, err := d.svc.GetCounterLevels(ctx, req.GetLabel())
	if err != nil {
		return nil, err
	}
	return &counterv1.GetCounterLevelsResp{StudentIds: ids}, nil
}

func (d *CounterServiceServer) RebuildCounter(ctx context.Context, req *counterv1.RebuildCounterReq) (*counterv1.RebuildCounterResp, error) {
	if err := d.svc.RebuildCounter(ctx); err != nil {
		return nil, err
	}
	return &counterv1.RebuildCounterResp{}, nil
}

func (d *CounterServiceServer) DecayCounter(ctx context.Context, req *counterv1.DecayCounterReq) (*counterv1.DecayCounterResp, error) {
	if err := d.svc.DecayCounter(ctx, req.GetStudentIds()); err != nil {
		return nil, err
	}
	return &counterv1.DecayCounterResp{}, nil
}

func (d *CounterServiceServer) BoostScores(ctx context.Context, req *counterv1.BoostScoresReq) (*counterv1.BoostScoresResp, error) {
	if err := d.svc.BoostScores(ctx, req.GetStudentIds()); err != nil {
		return nil, err
	}
	return &counterv1.BoostScoresResp{}, nil
}

func (d *CounterServiceServer) Register(server grpc.ServiceRegistrar) {
	counterv1.RegisterCounterServiceServer(server, d)
}