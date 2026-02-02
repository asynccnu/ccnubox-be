package grpc

import (
	"context"
	"fmt"

	"github.com/asynccnu/ccnubox-be/be-content/domain"
	contentv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/content/v1"
)

func (c *ContentServiceServer) GetWebsites(ctx context.Context, request *contentv1.GetWebsitesRequest) (*contentv1.GetWebsitesResponse, error) {
	webs, err := c.svcWebsite.GetList(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取常用网站列表失败: %w", err)
	}
	return &contentv1.GetWebsitesResponse{
		Websites: websiteDomains2GRPC(webs),
	}, nil
}

func (c *ContentServiceServer) SaveWebsite(ctx context.Context, request *contentv1.SaveWebsiteRequest) (*contentv1.SaveWebsiteResponse, error) {
	// 1. 调用 Service 执行保存
	err := c.svcWebsite.Save(ctx, &domain.Website{
		ID:          uint(request.Website.GetId()),
		Name:        request.Website.GetName(),
		Link:        request.Website.GetLink(),
		Image:       request.Website.GetImage(),
		Description: request.Website.GetDescription(),
	})
	if err != nil {
		return nil, fmt.Errorf("保存网站信息失败: %w", err)
	}

	return &contentv1.SaveWebsiteResponse{}, nil
}

func (c *ContentServiceServer) DelWebsite(ctx context.Context, request *contentv1.DelWebsiteRequest) (*contentv1.DelWebsiteResponse, error) {
	// 1. 调用 Service 执行删除
	err := c.svcWebsite.Del(ctx, request.GetId())
	if err != nil {
		return nil, fmt.Errorf("删除网站失败: %w", err)
	}

	return &contentv1.DelWebsiteResponse{}, nil
}

func websiteDomains2GRPC(webs []domain.Website) []*contentv1.Website {
	res := make([]*contentv1.Website, 0, len(webs))
	for _, w := range webs {
		res = append(res, &contentv1.Website{
			Id:          int64(w.ID),
			Name:        w.Name,
			Link:        w.Link,
			Image:       w.Image,
			Description: w.Description,
		})
	}
	return res
}
