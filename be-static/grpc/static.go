package grpc

import (
	"context"
	staticv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/static/v1"
	"github.com/asynccnu/ccnubox-be/be-static/domain"
	"github.com/asynccnu/ccnubox-be/be-static/service"
	"github.com/ecodeclub/ekit/slice"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

type StaticServiceServer struct {
	staticv1.UnimplementedStaticServiceServer
	svc service.StaticService
}

func NewStaticServiceServer(svc service.StaticService) *StaticServiceServer {
	return &StaticServiceServer{svc: svc}
}

func (s *StaticServiceServer) GetStaticByName(ctx context.Context, request *staticv1.GetStaticByNameRequest) (*staticv1.GetStaticByNameResponse, error) {
	static, err := s.svc.GetStaticByName(ctx, request.GetName())
	return &staticv1.GetStaticByNameResponse{
		Static: &staticv1.Static{
			Name:    static.Name,
			Content: static.Content,
			Labels:  static.Labels,
		},
	}, err
}

func (s *StaticServiceServer) SaveStatic(ctx context.Context, request *staticv1.SaveStaticRequest) (*staticv1.SaveStaticResponse, error) {
	err := s.svc.SaveStatic(ctx, domain.Static{
		Name:    request.GetStatic().GetName(),
		Content: request.GetStatic().GetContent(),
		Labels:  request.GetStatic().GetLabels(),
	})
	return &staticv1.SaveStaticResponse{}, err
}

func (s *StaticServiceServer) GetStaticsByLabels(ctx context.Context, request *staticv1.GetStaticsByLabelsRequest) (*staticv1.GetStaticsByLabelsResponse, error) {
	statics, err := s.svc.GetStaticsByLabels(ctx, request.GetLabels())
	return &staticv1.GetStaticsByLabelsResponse{
		Statics: slice.Map(statics, func(idx int, src domain.Static) *staticv1.Static {
			return &staticv1.Static{
				Name:    src.Name,
				Content: src.Content,
				Labels:  src.Labels,
			}
		}),
	}, err
}

func (s *StaticServiceServer) Register(server *grpc.Server) {
	staticv1.RegisterStaticServiceServer(server, s)
}
