package grpc

import (
	"context"
	bannerv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/banner/v1"
	"github.com/asynccnu/ccnubox-be/be-banner/domain"
	"github.com/asynccnu/ccnubox-be/be-banner/service"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

type BannerServiceServer struct {
	bannerv1.UnimplementedBannerServiceServer
	svc service.BannerService
}

func NewBannerServiceServer(svc service.BannerService) *BannerServiceServer {
	return &BannerServiceServer{svc: svc}
}

func (d *BannerServiceServer) GetBanners(ctx context.Context, request *bannerv1.GetBannersRequest) (*bannerv1.GetBannersResponse, error) {
	banners, err := d.svc.GetBanners(ctx)
	if err != nil {
		return nil, err
	}
	resp := &bannerv1.GetBannersResponse{}
	for _, b := range banners {
		resp.Banners = append(resp.Banners, &bannerv1.Banner{
			Id:          int64(b.ID),
			WebLink:     b.WebLink,
			PictureLink: b.PictureLink,
		})
	}
	return resp, nil
}

func (d *BannerServiceServer) SaveBanner(ctx context.Context, request *bannerv1.SaveBannerRequest) (*bannerv1.SaveBannerResponse, error) {
	err := d.svc.SaveBanner(ctx, &domain.Banner{
		WebLink:     request.WebLink,
		PictureLink: request.PictureLink,
	})
	if err != nil {
		return nil, err
	}
	return &bannerv1.SaveBannerResponse{}, nil
}

func (d *BannerServiceServer) DelBanner(ctx context.Context, request *bannerv1.DelBannerRequest) (*bannerv1.DelBannerResponse, error) {
	err := d.svc.DelBanner(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	return &bannerv1.DelBannerResponse{}, nil
}

// 注册为grpc服务
func (d *BannerServiceServer) Register(server *grpc.Server) {
	bannerv1.RegisterBannerServiceServer(server, d)
}
