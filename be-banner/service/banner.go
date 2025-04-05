package service

import (
	"context"
	bannerv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/banner/v1"
	"github.com/asynccnu/ccnubox-be/be-banner/domain"
	"github.com/asynccnu/ccnubox-be/be-banner/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/be-banner/pkg/logger"
	"github.com/asynccnu/ccnubox-be/be-banner/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-banner/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-banner/repository/model"
	"github.com/jinzhu/copier"
	"time"
)

type BannerService interface {
	GetBanners(ctx context.Context) ([]*domain.Banner, error)
	SaveBanner(ctx context.Context, req *domain.Banner) error
	DelBanner(ctx context.Context, id int64) error
}

type bannerService struct {
	dao   dao.BannerDAO
	cache cache.BannerCache
	l     logger.Logger
}

// 定义错误结构体
var (
	GET_BANNER_ERROR = func(err error) error {
		return errorx.New(bannerv1.ErrorGetBannerError("获取BANNER失败"), "dao", err)
	}

	DEL_BANNER_ERROR = func(err error) error {
		return errorx.New(bannerv1.ErrorDelBannerError("删除BANNER失败"), "dao", err)
	}

	SAVE_BANNER_ERROR = func(err error) error {
		return errorx.New(bannerv1.ErrorSaveBannerError("删除BANNER失败"), "dao", err)
	}
)

func NewBannerService(dao dao.BannerDAO, cache cache.BannerCache, l logger.Logger) BannerService {
	return &bannerService{dao: dao, cache: cache, l: l}
}

func (s *bannerService) GetBanners(ctx context.Context) ([]*domain.Banner, error) {
	// 尝试从缓存获取
	res, err := s.cache.GetBanners(ctx)
	if err == nil {
		return res, nil
	}
	s.l.Info("从缓存获取banner失败!", logger.FormatLog("cache", err)...)
	// 如果缓存中不存在则从数据库获取
	banners, err := s.dao.GetBanners(ctx)
	if err != nil {
		return []*domain.Banner{}, GET_BANNER_ERROR(err)
	}

	var resp []*domain.Banner
	err = copier.Copy(&resp, &banners)
	if err != nil {
		return nil, GET_BANNER_ERROR(err)
	}
	return resp, nil
}

func (s *bannerService) SaveBanner(ctx context.Context, req *domain.Banner) error {
	//查找数据库中是否存在,如果存在就
	banner, err := s.dao.FindBanner(ctx, int64(req.ID))
	if banner != nil {
		banner.WebLink = req.WebLink
		banner.PictureLink = req.PictureLink
	} else {
		banner = &model.Banner{
			WebLink:     req.WebLink,
			PictureLink: req.PictureLink,
		}
	}

	// 保存到数据库
	err = s.dao.SaveBanner(ctx, banner)
	if err != nil {
		return SAVE_BANNER_ERROR(err)
	}

	//异步写入缓存,牺牲一定的一致性
	go func() {
		ct, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		res, er := s.dao.GetBanners(ct)
		//类型转换
		var banners []*domain.Banner
		err := copier.Copy(&banners, &res)
		if err != nil {
			return
		}
		er = s.cache.SetBanners(ct, banners)
		if err != nil {
			s.l.Error("回写department资源失败", logger.FormatLog("cache", er)...)
		}
	}()

	return err
}

func (s *bannerService) DelBanner(ctx context.Context, id int64) error {
	err := s.dao.DelBanner(ctx, id)
	if err != nil {
		return DEL_BANNER_ERROR(err)
	}

	//异步写入缓存,牺牲一定的一致性
	go func() {
		ct, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		var banners []*domain.Banner
		res, er := s.dao.GetBanners(ct)
		//类型转换
		err := copier.Copy(&banners, &res)
		if err != nil {
			return
		}
		er = s.cache.SetBanners(ct, banners)
		if err != nil {
			s.l.Error("回写department资源失败", logger.FormatLog("cache", er)...)
		}

	}()

	// 更新缓存，或者立刻设置缓存，感觉不应该做异步
	return nil
}
