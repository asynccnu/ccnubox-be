package dao

import (
	"context"
	"github.com/asynccnu/ccnubox-be/be-banner/repository/model"
	"gorm.io/gorm"
)

type BannerDAO interface {
	GetBanners(ctx context.Context) ([]*model.Banner, error)
	SaveBanner(ctx context.Context, req *model.Banner) error
	DelBanner(ctx context.Context, Id int64) error
	FindBanner(ctx context.Context, Id int64) (*model.Banner, error)
}

type bannerDAO struct {
	gorm *gorm.DB
}

func NewMysqlBannerDAO(db *gorm.DB) BannerDAO {
	return &bannerDAO{gorm: db}
}

func (dao *bannerDAO) GetBanners(ctx context.Context) ([]*model.Banner, error) {
	var b []*model.Banner
	err := dao.gorm.WithContext(ctx).Find(&b).Error
	return b, err
}

func (dao *bannerDAO) SaveBanner(ctx context.Context, req *model.Banner) error {
	return dao.gorm.WithContext(ctx).Save(req).Error
}

func (dao *bannerDAO) FindBanner(ctx context.Context, Id int64) (*model.Banner, error) {
	b := model.Banner{}
	err := dao.gorm.WithContext(ctx).Where("id=?", Id).First(&b).Error
	return &b, err
}

func (dao *bannerDAO) DelBanner(ctx context.Context, Id int64) error {
	return dao.gorm.WithContext(ctx).Where("id=?", Id).Delete(&model.Banner{}).Error
}
