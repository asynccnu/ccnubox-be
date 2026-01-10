package grpc

import (
	"context"
	"github.com/asynccnu/ccnubox-be/be-content/domain"
	"github.com/asynccnu/ccnubox-be/be-content/pkg/errorx"
	contentv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/content/v1"
)

// 定义 Banner 相关的错误
var (
	GET_BANNERS_ERROR = errorx.FormatGRPCErrorFunc(contentv1.ErrorGetBannerError("获取 Banner 列表失败"))

	SAVE_BANNER_ERROR = errorx.FormatGRPCErrorFunc(contentv1.ErrorSaveBannerError("保存 Banner 失败"))

	DEL_BANNER_ERROR = errorx.FormatGRPCErrorFunc(contentv1.ErrorDelBannerError("删除 Banner 失败"))
)

func (c *ContentServiceServer) GetBanners(ctx context.Context, request *contentv1.GetBannersRequest) (*contentv1.GetBannersResponse, error) {
	banners, err := c.svcBanner.GetList(ctx)
	if err != nil {
		// 转换并返回带 Code 的错误
		return nil, GET_BANNERS_ERROR(err)
	}
	return &contentv1.GetBannersResponse{
		Banners: bannerDomains2GRPC(banners),
	}, nil
}

func (c *ContentServiceServer) SaveBanner(ctx context.Context, request *contentv1.SaveBannerRequest) (*contentv1.SaveBannerResponse, error) {
	err := c.svcBanner.Save(ctx, &domain.Banner{
		ID:          uint(request.Id),
		PictureLink: request.PictureLink,
		WebLink:     request.WebLink,
	})
	if err != nil {
		return nil, SAVE_BANNER_ERROR(err)
	}
	return &contentv1.SaveBannerResponse{}, nil
}

func (c *ContentServiceServer) DelBanner(ctx context.Context, request *contentv1.DelBannerRequest) (*contentv1.DelBannerResponse, error) {
	err := c.svcBanner.Del(ctx, request.Id)
	if err != nil {
		return nil, DEL_BANNER_ERROR(err)
	}
	return &contentv1.DelBannerResponse{}, nil
}

func bannerDomains2GRPC(banners []domain.Banner) []*contentv1.Banner {
	res := make([]*contentv1.Banner, 0, len(banners))
	for _, c := range banners {
		res = append(res, &contentv1.Banner{
			Id:          int64(c.ID),
			WebLink:     c.WebLink,
			PictureLink: c.PictureLink,
		})
	}
	return res
}
