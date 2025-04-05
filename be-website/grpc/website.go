package grpc

import (
	"context"
	websitev1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/website/v1"
	"github.com/asynccnu/ccnubox-be/be-website/domain"
	"github.com/asynccnu/ccnubox-be/be-website/service"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/jinzhu/copier"
	"gorm.io/gorm"
)

type WebsiteServiceServer struct {
	websitev1.UnimplementedWebsiteServiceServer
	svc service.WebsiteService
}

func NewWebsiteServiceServer(svc service.WebsiteService) *WebsiteServiceServer {
	return &WebsiteServiceServer{svc: svc}
}

func (d *WebsiteServiceServer) GetWebsites(ctx context.Context, request *websitev1.GetWebsitesRequest) (*websitev1.GetWebsitesResponse, error) {
	websites, err := d.svc.GetWebsites(ctx)
	if err != nil {
		return nil, err
	}
	var resp []*websitev1.Website
	err = copier.Copy(&resp, websites)
	if err != nil {

		return nil, err
	}
	return &websitev1.GetWebsitesResponse{Websites: resp}, nil
}

func (d *WebsiteServiceServer) SaveWebsite(ctx context.Context, request *websitev1.SaveWebsiteRequest) (*websitev1.SaveWebsiteResponse, error) {
	return &websitev1.SaveWebsiteResponse{}, d.svc.SaveWebsite(ctx, convertToDomain(request.Website))
}

func (d *WebsiteServiceServer) DelWebsite(ctx context.Context, request *websitev1.DelWebsiteRequest) (*websitev1.DelWebsiteResponse, error) {
	return &websitev1.DelWebsiteResponse{}, d.svc.DelWebsite(ctx, uint(request.Id))
}

// 注册为grpc服务
func (d *WebsiteServiceServer) Register(server *grpc.Server) {
	websitev1.RegisterWebsiteServiceServer(server, d)
}

func convertToV(website domain.Website) *websitev1.Website {
	return &websitev1.Website{
		Id:          int64(website.ID),
		Name:        website.Name,
		Link:        website.Link,
		Image:       website.Image,
		Description: website.Description,
	}
}

func convertToDomain(website *websitev1.Website) *domain.Website {
	return &domain.Website{
		Name:        website.Name,
		Link:        website.Link,
		Description: website.Description,
		Image:       website.Image,
		Model:       gorm.Model{ID: uint(website.Id)},
	}
}
