package service

import (
	"context"
	websitev1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/website/v1"
	"github.com/asynccnu/ccnubox-be/be-website/domain"
	"github.com/asynccnu/ccnubox-be/be-website/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/be-website/pkg/logger"
	"github.com/asynccnu/ccnubox-be/be-website/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-website/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-website/repository/model"
	"github.com/jinzhu/copier"
	"time"
)

type WebsiteService interface {
	GetWebsites(ctx context.Context) ([]*domain.Website, error)
	SaveWebsite(ctx context.Context, req *domain.Website) error
	DelWebsite(ctx context.Context, id uint) error
}

type websiteService struct {
	dao   dao.WebsiteDAO
	cache cache.WebsiteCache
	l     logger.Logger
}

// 定义错误函数
var (
	GET_WEBSITE_ERROR = func(err error) error {
		return errorx.New(websitev1.ErrorGetWebsiteError("获取整合信息失败"), "dao", err)
	}

	DEL_WEBSITE_ERROR = func(err error) error {
		return errorx.New(websitev1.ErrorDelWebsiteError("删除整合信息失败"), "dao", err)
	}

	SAVE_WEBSITE_ERROR = func(err error) error {
		return errorx.New(websitev1.ErrorSaveWebsiteError("保存整合信息失败"), "dao", err)
	}
)

func NewWebsiteService(dao dao.WebsiteDAO, cache cache.WebsiteCache, l logger.Logger) WebsiteService {
	return &websiteService{dao: dao, cache: cache, l: l}
}

func (s *websiteService) GetWebsites(ctx context.Context) ([]*domain.Website, error) {
	//尝试从缓存获取,如果获取直接返回结果
	res, err := s.cache.GetWebsites(ctx)
	if err == nil {
		return res, nil
	}
	s.l.Error("从缓存获取website失败", logger.FormatLog("cache", err)...)

	var resp []*domain.Website
	//如果缓存中不存在则从数据库获取
	webs, err := s.dao.GetWebsites(ctx)
	if err != nil {
		return []*domain.Website{}, GET_WEBSITE_ERROR(err)
	}

	//类型转换
	err = copier.Copy(&resp, webs)
	if err != nil {
		return []*domain.Website{}, GET_WEBSITE_ERROR(err)
	}

	return resp, nil
}

func (s *websiteService) SaveWebsite(ctx context.Context, req *domain.Website) error {
	website, err := s.dao.FindWebsite(ctx, req.ID)
	if website != nil {
		website.Name = req.Name
		website.Link = req.Link
		website.Description = req.Description
		website.Image = req.Image
	} else {
		website = s.toEntity(req)
	}

	//保存到数据库
	err = s.dao.SaveWebsite(ctx, website)
	if err != nil {
		return SAVE_WEBSITE_ERROR(err)
	}

	//异步写入缓存,牺牲一定的一致性
	go func() {
		ct, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		resp, err := s.dao.GetWebsites(ctx)
		if err != nil {
			s.l.Error("回写websites字段时从dao层中获取失败", logger.FormatLog("cache", err)...)
		}
		var websites []*domain.Website
		err = copier.Copy(resp, websites)
		if err != nil {
			return
		}
		err = s.cache.SetWebsites(ct, websites)
		if err != nil {
			s.l.Error("回写websites资源失败", logger.FormatLog("cache", err)...)
		}
	}()

	return nil
}

func (s *websiteService) DelWebsite(ctx context.Context, id uint) error {
	err := s.dao.DelWebsite(ctx, id)
	if err != nil {
		return DEL_WEBSITE_ERROR(err)
	}

	//异步写入缓存,牺牲一定的一致性
	go func() {
		ct, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		resp, err := s.dao.GetWebsites(ctx)
		if err != nil {
			s.l.Error("回写websites字段时从dao层中获取失败", logger.FormatLog("cache", err)...)
		}
		var websites []*domain.Website
		err = copier.Copy(resp, websites)
		if err != nil {
			return
		}
		err = s.cache.SetWebsites(ct, websites)
		if err != nil {
			s.l.Error("回写websites资源失败", logger.FormatLog("cache", err)...)
		}
	}()

	return nil
}

func (s *websiteService) toDomain(u *model.Website) *domain.Website {
	return &domain.Website{
		Name:        u.Name,
		Link:        u.Link,
		Description: u.Description,
		Image:       u.Image,
		Model:       u.Model,
	}
}

func (s *websiteService) toEntity(u *domain.Website) *model.Website {
	return &model.Website{
		Name:        u.Name,
		Link:        u.Link,
		Description: u.Description,
		Image:       u.Image,
		Model:       u.Model,
	}
}
