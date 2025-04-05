package grpc

import (
	"context"
	counterv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/counter/v1"
	"github.com/asynccnu/ccnubox-be/be-counter/domain"
	"github.com/asynccnu/ccnubox-be/be-counter/service"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

type CounterServiceServer struct {
	counterv1.UnimplementedCounterServiceServer
	svc service.CounterService
}

func NewCounterServiceServer(svc service.CounterService) *CounterServiceServer {
	return &CounterServiceServer{svc: svc}
}

func (d *CounterServiceServer) AddCounter(ctx context.Context, request *counterv1.AddCounterReq) (*counterv1.AddCounterResp, error) {
	err := d.svc.AddCounter(ctx, request.GetStudentId())
	if err != nil {
		return nil, err
	}
	return &counterv1.AddCounterResp{}, nil
}

func (d *CounterServiceServer) ChangeCounterLevels(ctx context.Context, request *counterv1.ChangeCounterLevelsReq) (*counterv1.ChangeCounterLevelsResp, error) {
	err := d.svc.ChangeCounterLevels(ctx, domain.ChangeCounterLevels{
		StudentIds: request.StudentIds,
		IsReduce:   request.IsReduce,
		Steps:      request.Step,
	})
	if err != nil {
		return nil, err
	}
	return &counterv1.ChangeCounterLevelsResp{}, nil
}

func (d *CounterServiceServer) GetCounterLevels(ctx context.Context, request *counterv1.GetCounterLevelsReq) (*counterv1.GetCounterLevelsResp, error) {
	levels, err := d.svc.GetCounterLevels(ctx, request.GetLabel())
	if err != nil {
		return nil, err
	}
	return &counterv1.GetCounterLevelsResp{StudentIds: levels}, nil
}

func (d *CounterServiceServer) ClearCounterLevels(ctx context.Context, req *counterv1.ClearCounterLevelsReq) (*counterv1.ClearCounterLevelsResp, error) {
	err := d.svc.ClearCounterLevels(ctx)
	if err != nil {
		return nil, err
	}
	return &counterv1.ClearCounterLevelsResp{}, nil
}

// 注册为grpc服务
func (d *CounterServiceServer) Register(server *grpc.Server) {
	counterv1.RegisterCounterServiceServer(server, d)
}
