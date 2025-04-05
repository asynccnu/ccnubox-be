package dao

import (
	"context"
	"github.com/asynccnu/ccnubox-be/be-website/repository/model"
	"gorm.io/gorm"
)

type WebsiteDAO interface {
	GetWebsites(ctx context.Context) ([]*model.Website, error)
	SaveWebsite(ctx context.Context, req *model.Website) error
	DelWebsite(ctx context.Context, req uint) error
	FindWebsite(ctx context.Context, Id uint) (*model.Website, error)
}

type websiteDAO struct {
	gorm *gorm.DB
}

func NewMysqlWebsiteDAO(db *gorm.DB) WebsiteDAO {
	return &websiteDAO{gorm: db}
}

func (dao *websiteDAO) GetWebsites(ctx context.Context) ([]*model.Website, error) {
	var s []*model.Website
	err := dao.gorm.WithContext(ctx).Table("websites").Find(&s).Error
	return s, err
}

func (dao *websiteDAO) SaveWebsite(ctx context.Context, req *model.Website) error {
	return dao.gorm.WithContext(ctx).Table("websites").Save(req).Error
}

func (dao *websiteDAO) FindWebsite(ctx context.Context, Id uint) (*model.Website, error) {
	d := model.Website{}
	err := dao.gorm.WithContext(ctx).Table("websites").Where("id=?", Id).First(&d).Error
	return &d, err
}

func (dao *websiteDAO) DelWebsite(ctx context.Context, id uint) error {
	return dao.gorm.WithContext(ctx).Table("websites").Where("id=?", id).Delete(&model.Website{}).Error
}
