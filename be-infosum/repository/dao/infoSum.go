package dao

import (
	"context"
	"github.com/asynccnu/ccnubox-be/be-infosum/repository/model"
	"gorm.io/gorm"
)

type InfoSumDAO interface {
	GetInfoSums(ctx context.Context) ([]*model.InfoSum, error)
	SaveInfoSum(ctx context.Context, req *model.InfoSum) error
	DelInfoSum(ctx context.Context, req uint) error
	FindInfoSum(ctx context.Context, Id uint) (*model.InfoSum, error)
}

type infoSumDAO struct {
	gorm *gorm.DB
}

func NewMysqlInfoSumDAO(db *gorm.DB) InfoSumDAO {
	return &infoSumDAO{gorm: db}
}

func (dao infoSumDAO) GetInfoSums(ctx context.Context) ([]*model.InfoSum, error) {
	var s []*model.InfoSum
	err := dao.gorm.WithContext(ctx).Find(&s).Error
	return s, err
}

func (dao infoSumDAO) SaveInfoSum(ctx context.Context, req *model.InfoSum) error {
	return dao.gorm.WithContext(ctx).Save(req).Error
}

func (dao infoSumDAO) FindInfoSum(ctx context.Context, Id uint) (*model.InfoSum, error) {
	d := model.InfoSum{}
	err := dao.gorm.WithContext(ctx).Where("id=?", Id).First(&d).Error
	return &d, err
}

func (dao infoSumDAO) DelInfoSum(ctx context.Context, id uint) error {
	return dao.gorm.WithContext(ctx).Where("id=?", id).Delete(&model.InfoSum{}).Error
}
