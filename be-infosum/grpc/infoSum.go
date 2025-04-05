package grpc

import (
	"context"
	InfoSumv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/infoSum/v1"
	"github.com/asynccnu/ccnubox-be/be-infosum/domain"
	"github.com/asynccnu/ccnubox-be/be-infosum/service"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/jinzhu/copier"
	"gorm.io/gorm"
)

type InfoSumServiceServer struct {
	InfoSumv1.UnimplementedInfoSumServiceServer
	svc service.InfoSumService
}

func NewInfoSumServiceServer(svc service.InfoSumService) *InfoSumServiceServer {
	return &InfoSumServiceServer{svc: svc}
}

func (d *InfoSumServiceServer) GetInfoSums(ctx context.Context, request *InfoSumv1.GetInfoSumsRequest) (*InfoSumv1.GetInfoSumsResponse, error) {
	InfoSums, err := d.svc.GetInfoSums(ctx)
	if err != nil {
		return nil, err
	}

	var resp []*InfoSumv1.InfoSum
	err = copier.Copy(&resp, InfoSums)
	if err != nil {
		return nil, err
	}
	return &InfoSumv1.GetInfoSumsResponse{InfoSums: resp}, nil
}

func (d *InfoSumServiceServer) SaveInfoSum(ctx context.Context, request *InfoSumv1.SaveInfoSumRequest) (*InfoSumv1.SaveInfoSumResponse, error) {
	err := d.svc.SaveInfoSum(ctx, convertToDomain(request.InfoSum))
	if err != nil {
		return nil, err
	}
	return &InfoSumv1.SaveInfoSumResponse{}, nil
}

func (d *InfoSumServiceServer) DelInfoSum(ctx context.Context, request *InfoSumv1.DelInfoSumRequest) (*InfoSumv1.DelInfoSumResponse, error) {
	return &InfoSumv1.DelInfoSumResponse{}, d.svc.DelInfoSum(ctx, uint(request.Id))
}

// 注册为grpc服务
func (d *InfoSumServiceServer) Register(server *grpc.Server) {
	InfoSumv1.RegisterInfoSumServiceServer(server, d)
}

func convertToV(InfoSum domain.InfoSum) *InfoSumv1.InfoSum {
	return &InfoSumv1.InfoSum{
		Id:          int64(InfoSum.ID),
		Name:        InfoSum.Name,
		Link:        InfoSum.Link,
		Image:       InfoSum.Image,
		Description: InfoSum.Description,
	}
}

func convertToDomain(InfoSum *InfoSumv1.InfoSum) *domain.InfoSum {
	return &domain.InfoSum{
		Name:        InfoSum.Name,
		Link:        InfoSum.Link,
		Description: InfoSum.Description,
		Image:       InfoSum.Image,
		Model:       gorm.Model{ID: uint(InfoSum.Id)},
	}
}
